// Copyright 2022 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strings"
)

const (
	trimPrefix   = "gitea_"
	sourceFolder = "options/locales/"
)

// returns list of locales, still containing the file extension!
func generate_locale_list() []string {
	localeFiles, _ := os.ReadDir(sourceFolder)
	locales := []string{}
	for _, localeFile := range localeFiles {
		if !localeFile.IsDir() && strings.HasPrefix(localeFile.Name(), trimPrefix) {
			locales = append(locales, strings.TrimPrefix(localeFile.Name(), trimPrefix))
		}
	}
	return locales
}

// replace all occurrences of Gitea with Forgejo
func renameGiteaForgejo(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	replacements := []string{
		"Gitea", "Forgejo",
		"https://docs.gitea.com/installation/install-from-binary", "https://forgejo.org/download/#installation-from-binary",
		"https://github.com/go-gitea/gitea/tree/master/docker", "https://forgejo.org/download/#container-image",
		"https://docs.gitea.com/installation/install-from-package", "https://forgejo.org/download",
		"https://code.gitea.io/gitea", "https://forgejo.org/download",
		"code.gitea.io/gitea", "Forgejo",
		`<a href="https://github.com/go-gitea/gitea/issues" target="_blank">GitHub</a>`, `<a href="https://codeberg.org/forgejo/forgejo/issues" target="_blank">Codeberg</a>`,
		"https://github.com/go-gitea/gitea", "https://codeberg.org/forgejo/forgejo",
		"https://blog.gitea.io", "https://forgejo.org/news",
		"https://docs.gitea.com/usage/protected-tags", "https://forgejo.org/docs/latest/user/protection/#protected-tags",
		"https://docs.gitea.com/usage/webhooks", "https://forgejo.org/docs/latest/user/webhooks/",
	}
	replacer := strings.NewReplacer(replacements...)
	replaced := make(map[string]bool, len(replacements)/2)
	count_replaced := func(original string) {
		for i := 0; i < len(replacements); i += 2 {
			if strings.Contains(original, replacements[i]) {
				replaced[replacements[i]] = true
			}
		}
	}

	out := make([]byte, 0, 1024)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "license_desc=") {
			line = strings.Replace(line, "GitHub", "Forgejo", 1)
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			out = append(out, []byte(line+"\n")...)
		} else if strings.HasPrefix(line, "settings.web_hook_name_gitea") {
			out = append(out, []byte(line+"\n")...)
			out = append(out, []byte("settings.web_hook_name_forgejo = Forgejo\n")...)
		} else if strings.HasPrefix(line, "migrate.gitea.description") {
			re := regexp.MustCompile(`(.*Gitea)`)
			out = append(out, []byte(re.ReplaceAllString(line, "${1}/Forgejo")+"\n")...)
		} else {
			count_replaced(line)
			out = append(out, []byte(replacer.Replace(line)+"\n")...)
		}
	}
	file.Close()
	if strings.HasSuffix(filename, "gitea_en-US.ini") {
		for i := 0; i < len(replacements); i += 2 {
			if replaced[replacements[i]] == false {
				log.Fatalf("%s was never used to replace something in %s, it is obsolete and must be updated", replacements[i], filename)
			}
		}
	}
	return out
}

func main() {
	d := os.Args[1]
	files, err := os.ReadDir(d)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		p := d + "/" + f.Name()
		os.WriteFile(p, renameGiteaForgejo(p), 0o644)
	}
}
