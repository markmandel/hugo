// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hugofs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gohugoio/hugo/langs"
	"github.com/spf13/viper"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/require"
)

// TODO(bep) mod
// tc-lib-color/class-Com.Tecnick.Color.Css and class-Com.Tecnick.Color.sv.Css

func TestLanguageRootMapping(t *testing.T) {
	assert := require.New(t)
	v := viper.New()
	v.Set("contentDir", "content")

	fs := afero.NewMemMapFs()

	testfile := "test.txt"

	assert.NoError(afero.WriteFile(fs, filepath.Join("themes/a/mysvblogcontent", testfile), []byte("some sv blog content"), 0755))
	assert.NoError(afero.WriteFile(fs, filepath.Join("themes/a/myenblogcontent", testfile), []byte("some en blog content in a"), 0755))

	assert.NoError(afero.WriteFile(fs, filepath.Join("themes/a/mysvdocs", testfile), []byte("some sv docs content"), 0755))

	assert.NoError(afero.WriteFile(fs, filepath.Join("themes/b/myenblogcontent", testfile), []byte("some en content"), 0755))

	bfs := DecorateBasePathFs(afero.NewBasePathFs(fs, "themes").(*afero.BasePathFs))

	rfs, err := NewRootMappingFs(bfs,
		RootMapping{
			From: "content/blog",      // Virtual path, first element is one of content, static, layouts etc.
			To:   "a/mysvblogcontent", // Real path
			Meta: FileMeta{"lang": "sv"},
		},
		RootMapping{
			From: "content/blog",
			To:   "a/myenblogcontent",
			Meta: FileMeta{"lang": "en"},
		},
		RootMapping{
			From: "content/docs",
			To:   "a/mysvdocs",
			Meta: FileMeta{"lang": "sv"},
		},
	)

	assert.NoError(err)

	dirs, err := rfs.Dirs("content/blog")
	assert.NoError(err)
	assert.Equal(2, len(dirs))

}

func TestLanguageMeta(t *testing.T) {
	assert := require.New(t)
	v := viper.New()
	v.Set("contentDir", "content")

	languages := langs.Languages{
		langs.NewLanguage("en", v),
		langs.NewLanguage("sv", v),
		langs.NewLanguage("nn", v),
	}

	fs := NewBaseFileDecorator(afero.NewMemMapFs())

	type testConfig struct {
		lang string
		to   string // real path

	}

	testConfigs := []testConfig{
		testConfig{"en", "themes/myenblog"},
		testConfig{"sv", "themes/mysvblog"},
		testConfig{"nn", "themes/mynnblog"},
	}

	writeFile := func(cfg testConfig, filename string) {
		assert.NoError(afero.WriteFile(fs, filepath.Join(cfg.to, filename), []byte(fmt.Sprintf("%s in %s", filename, cfg.lang)), 0755))
	}

	// Fully translated
	for _, cfg := range testConfigs {
		writeFile(cfg, "fullytranslated/index.md")
		writeFile(cfg, "fullytranslated/data.json")
	}

	// Partly translated
	for i, cfg := range testConfigs {
		if i == 0 {
			continue
		}
		writeFile(cfg, "partlytranslated/data.json")
	}

	var rootMappings []RootMapping

	for _, cfg := range testConfigs {
		rootMappings = append(rootMappings, RootMapping{
			From: "content/blog",
			To:   cfg.to,
			Meta: FileMeta{"lang": cfg.lang},
		})
	}

	rfs, err := NewRootMappingFs(fs, rootMappings...)

	assert.NoError(err)

	dirs, err := rfs.Dirs("content/blog")
	assert.NoError(err)
	assert.Equal(3, len(dirs))

	langFs, err := NewLanguageFs(languages.AsSet(), dirs...)
	assert.NoError(err)
	assert.NotNil(langFs)

	fullytranslated, err := langFs.Open("fullytranslated")
	assert.NoError(err)

	fis, err := fullytranslated.Readdir(-1)
	assert.NoError(err)
	assert.Equal(6, len(fis))

	pickOne := func(fis []os.FileInfo, matches func(fim FileMetaInfo) bool) FileMetaInfo {
		for _, fi := range fis {
			fim := fi.(FileMetaInfo)
			if matches(fim) {
				return fim
			}
		}
		assert.Fail("fi not found")
		return nil
	}

	fi := pickOne(fis, func(fim FileMetaInfo) bool { return fim.Name() == "index.md" && fim.Meta().Lang() == "sv" })

	meta := fi.Meta()
	assert.Equal([]string{"en", "nn", "sv"}, meta.Translations())

	partlytranslated, err := langFs.Open("partlytranslated")
	assert.NoError(err)

	fis, err = partlytranslated.Readdir(-1)
	assert.NoError(err)
	assert.Equal(2, len(fis))

	fi = pickOne(fis, func(fim FileMetaInfo) bool { return fim.Meta().Lang() == "nn" })
	meta = fi.Meta()
	assert.Equal([]string{"nn", "sv"}, meta.Translations())
}
