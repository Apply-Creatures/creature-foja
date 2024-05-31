// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/auth/password"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"

	"github.com/google/uuid"
)

// ProcessLikeActivity receives a ForgeLike activity and does the following:
// Validation of the activity
// Creation of a (remote) federationHost if not existing
// Creation of a forgefed Person if not existing
// Validation of incoming RepositoryID against Local RepositoryID
// Star the repo if it wasn't already stared
// Do some mitigation against out of order attacks
func ProcessLikeActivity(ctx context.Context, form any, repositoryID int64) (int, string, error) {
	activity := form.(*fm.ForgeLike)
	if res, err := validation.IsValid(activity); !res {
		return http.StatusNotAcceptable, "Invalid activity", err
	}
	log.Info("Activity validated:%v", activity)

	// parse actorID (person)
	actorURI := activity.Actor.GetID().String()
	log.Info("actorURI was: %v", actorURI)
	federationHost, err := GetFederationHostForURI(ctx, actorURI)
	if err != nil {
		return http.StatusInternalServerError, "Wrong FederationHost", err
	}
	if !activity.IsNewer(federationHost.LatestActivity) {
		return http.StatusNotAcceptable, "Activity out of order.", fmt.Errorf("Activity already processed")
	}
	actorID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid PersonID", err
	}
	log.Info("Actor accepted:%v", actorID)

	// parse objectID (repository)
	objectID, err := fm.NewRepositoryID(activity.Object.GetID().String(), string(forgefed.ForgejoSourceType))
	if err != nil {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	if objectID.ID != fmt.Sprint(repositoryID) {
		return http.StatusNotAcceptable, "Invalid objectId", err
	}
	log.Info("Object accepted:%v", objectID)

	// Check if user already exists
	user, _, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		return http.StatusInternalServerError, "Searching for user failed", err
	}
	if user != nil {
		log.Info("Found local federatedUser: %v", user)
	} else {
		user, _, err = CreateUserFromAP(ctx, actorID, federationHost.ID)
		if err != nil {
			return http.StatusInternalServerError, "Error creating federatedUser", err
		}
		log.Info("Created federatedUser from ap: %v", user)
	}
	log.Info("Got user:%v", user.Name)

	// execute the activity if the repo was not stared already
	alreadyStared := repo.IsStaring(ctx, user.ID, repositoryID)
	if !alreadyStared {
		err = repo.StarRepo(ctx, user.ID, repositoryID, true)
		if err != nil {
			return http.StatusNotAcceptable, "Error staring", err
		}
	}
	federationHost.LatestActivity = activity.StartTime
	err = forgefed.UpdateFederationHost(ctx, federationHost)
	if err != nil {
		return http.StatusNotAcceptable, "Error updating federatedHost", err
	}

	return 0, "", nil
}

func CreateFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*forgefed.FederationHost, error) {
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, err
	}
	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return nil, err
	}
	nodeInfoWellKnown, err := forgefed.NewNodeInfoWellKnown(body)
	if err != nil {
		return nil, err
	}
	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return nil, err
	}
	nodeInfo, err := forgefed.NewNodeInfo(body)
	if err != nil {
		return nil, err
	}
	result, err := forgefed.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return nil, err
	}
	err = forgefed.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func GetFederationHostForURI(ctx context.Context, actorURI string) (*forgefed.FederationHost, error) {
	log.Info("Input was: %v", actorURI)
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := forgefed.FindFederationHostByFqdn(ctx, rawActorID.Host)
	if err != nil {
		return nil, err
	}
	if federationHost == nil {
		result, err := CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = result
	}
	return federationHost, nil
}

func CreateUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user.NewActionsUser()
	client, err := activitypub.NewClient(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, nil, err
	}

	body, err := client.GetBody(personID.AsURI())
	if err != nil {
		return nil, nil, err
	}

	person := fm.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return nil, nil, err
	}
	if res, err := validation.IsValid(person); !res {
		return nil, nil, err
	}
	log.Info("Fetched valid person:%q", person)

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return nil, nil, err
	}
	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	fullName := person.Name.String()
	if len(person.Name) == 0 {
		fullName = name
	}
	password, err := password.Generate(32)
	if err != nil {
		return nil, nil, err
	}
	newUser := user.User{
		LowerName:                    strings.ToLower(name),
		Name:                         name,
		FullName:                     fullName,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    loginName,
		Type:                         user.UserTypeRemoteUser,
		IsAdmin:                      false,
		NormalizedFederatedURI:       personID.AsURI(),
	}
	federatedUser := user.FederatedUser{
		ExternalID:       personID.ID,
		FederationHostID: federationHostID,
	}
	err = user.CreateFederatedUser(ctx, &newUser, &federatedUser)
	if err != nil {
		return nil, nil, err
	}
	log.Info("Created federatedUser:%q", federatedUser)

	return &newUser, &federatedUser, nil
}

// Create or update a list of FollowingRepo structs
func StoreFollowingRepoList(ctx context.Context, localRepoID int64, followingRepoList []string) (int, string, error) {
	followingRepos := make([]*repo.FollowingRepo, 0, len(followingRepoList))
	for _, uri := range followingRepoList {
		federationHost, err := GetFederationHostForURI(ctx, uri)
		if err != nil {
			return http.StatusInternalServerError, "Wrong FederationHost", err
		}
		followingRepoID, err := fm.NewRepositoryID(uri, string(federationHost.NodeInfo.SoftwareName))
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepo, err := repo.NewFollowingRepo(localRepoID, followingRepoID.ID, federationHost.ID, uri)
		if err != nil {
			return http.StatusNotAcceptable, "Invalid federated repo", err
		}
		followingRepos = append(followingRepos, &followingRepo)
	}

	if err := repo.StoreFollowingRepos(ctx, localRepoID, followingRepos); err != nil {
		return 0, "", err
	}

	return 0, "", nil
}

func DeleteFollowingRepos(ctx context.Context, localRepoID int64) error {
	return repo.StoreFollowingRepos(ctx, localRepoID, []*repo.FollowingRepo{})
}
