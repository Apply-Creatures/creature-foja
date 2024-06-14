// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &label{}

type label struct {
	common

	forgejoLabel *issues_model.Label
}

func (o *label) SetNative(label any) {
	o.forgejoLabel = label.(*issues_model.Label)
}

func (o *label) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoLabel.ID)
}

func (o *label) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *label) ToFormat() f3.Interface {
	if o.forgejoLabel == nil {
		return o.NewFormat()
	}
	return &f3.Label{
		Common:      f3.NewCommon(fmt.Sprintf("%d", o.forgejoLabel.ID)),
		Name:        o.forgejoLabel.Name,
		Color:       o.forgejoLabel.Color,
		Description: o.forgejoLabel.Description,
	}
}

func (o *label) FromFormat(content f3.Interface) {
	label := content.(*f3.Label)
	o.forgejoLabel = &issues_model.Label{
		ID:          f3_util.ParseInt(label.GetID()),
		Name:        label.Name,
		Description: label.Description,
		Color:       label.Color,
	}
}

func (o *label) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())
	id := f3_util.ParseInt(string(node.GetID()))

	label, err := issues_model.GetLabelInRepoByID(ctx, project, id)
	if issues_model.IsErrRepoLabelNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("label %v %w", id, err))
	}
	o.forgejoLabel = label
	return true
}

func (o *label) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoLabel.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoLabel.ID).Cols("name", "description").Update(o.forgejoLabel); err != nil {
		panic(fmt.Errorf("UpdateLabelCols: %v %v", o.forgejoLabel, err))
	}
}

func (o *label) Put(ctx context.Context) generic.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	o.forgejoLabel.RepoID = f3_tree.GetProjectID(o.GetNode())
	if err := issues_model.NewLabel(ctx, o.forgejoLabel); err != nil {
		panic(err)
	}
	o.Trace("label created %d", o.forgejoLabel.ID)
	return generic.NodeID(fmt.Sprintf("%d", o.forgejoLabel.ID))
}

func (o *label) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())

	if err := issues_model.DeleteLabel(ctx, project, o.forgejoLabel.ID); err != nil {
		panic(err)
	}
}

func newLabel() generic.NodeDriverInterface {
	return &label{}
}
