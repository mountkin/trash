package conf

import (
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	yaml "github.com/cloudfoundry-incubator/candiedyaml"
)

type Conf struct {
	Package     string   `yaml:"package,omitempty"`
	Imports     []Import `yaml:"import,omitempty"`
	Excludes    []string `yaml:"exclude,omitempty"`
	IgnoredTags []string `yaml:"ignored_tags,omitempty"`
	IgnoredPkgs []string `yaml:"ignored_pkgs,omitempty"`
	NativeOnly  bool     `yaml:"native_only,omitempty"`
	importMap   map[string]Import
	confFile    string
	yamlType    bool
}

type Import struct {
	Package string `yaml:"package,omitempty"`
	Version string `yaml:"version,omitempty"`
	Repo    string `yaml:"repo,omitempty"`
	Options
}

type Options struct {
	Transitive bool `yaml:"transitive,omitempty"`
	Staging    bool `yaml:"staging,omitempty"`
}

func Parse(path string) (*Conf, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	trashConf := &Conf{confFile: path}
	err = yaml.NewDecoder(file).Decode(trashConf)
	if err != nil {
		return nil, err
	}

	trashConf.yamlType = true
	trashConf.Dedupe()
	if len(trashConf.IgnoredTags) == 0 {
		trashConf.IgnoredTags = []string{"ignore"}
	} else {
		trashConf.IgnoredTags = append(trashConf.IgnoredTags, "ignore")
	}
	trashConf.IgnoredTags = strUnique(trashConf.IgnoredTags)

	for i, s := range trashConf.IgnoredPkgs {
		trashConf.IgnoredPkgs[i] = strings.Trim(s, "/")
	}
	return trashConf, nil
}

func strUnique(src []string) []string {
	ret := []string{}
	m := make(map[string]bool)
	for _, s := range src {
		if !m[s] {
			m[s] = true
			ret = append(ret, s)
		}
	}
	return ret
}

// Dedupe deletes duplicates and sorts the imports
func (t *Conf) Dedupe() {
	t.importMap = map[string]Import{}
	for _, i := range t.Imports {
		if _, ok := t.importMap[i.Package]; ok {
			logrus.Debugf("Package '%s' has duplicates (in %s)", i.Package, t.confFile)
			continue
		}
		t.importMap[i.Package] = i
	}
	ps := make([]string, 0, len(t.importMap))
	for p := range t.importMap {
		ps = append(ps, p)
	}
	sort.Strings(ps)
	imports := make([]Import, 0, len(t.importMap))
	for _, p := range ps {
		imports = append(imports, t.importMap[p])
	}
	t.Imports = imports
}

func (t *Conf) Get(pkg string) (Import, bool) {
	i, ok := t.importMap[pkg]
	return i, ok
}

func (t *Conf) ConfFile() string {
	return t.confFile
}

func (t *Conf) Dump(path string) error {
	fp, err := ioutil.TempFile("", "vndr")
	if err != nil {
		return err
	}
	if err := yaml.NewEncoder(fp).Encode(t); err != nil {
		fp.Close()
		return err
	}
	fp.Close()
	return os.Rename(fp.Name(), path)
}
