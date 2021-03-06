package repotest

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/cfg"
	"github.com/simplesurance/baur/v1/internal/testutils/dbtest"
	"github.com/simplesurance/baur/v1/internal/testutils/fstest"
)

func (r *Repo) CreateAppWithoutTasks(t *testing.T) *cfg.App {
	t.Helper()

	appName := "appWithoutIO"

	app := cfg.App{
		Name: appName,
	}

	appDir := filepath.Join(r.Dir, appName)

	if err := os.Mkdir(appDir, 0775); err != nil {
		t.Fatal(err)
	}

	if err := app.ToFile(filepath.Join(appDir, baur.AppCfgFile)); err != nil {
		t.Fatalf("writing app cfg file failed: %s", err)
	}

	r.AppCfgs = append(r.AppCfgs, &app)

	return &app
}

func (r *Repo) CreateSimpleApp(t *testing.T) *cfg.App {
	t.Helper()

	appName := "simpleApp"

	app := cfg.App{
		Name: appName,
		Tasks: []*cfg.Task{
			{
				Name:    "build",
				Command: []string{"sh", "./build.sh"},
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths: []string{"build.sh", "output_content.txt"},
						},
					},
				},
				Output: cfg.Output{
					File: []cfg.FileOutput{
						{
							Path: "output",
							FileCopy: cfg.FileCopy{
								Path: r.FilecopyArtifactDir,
							},
						},
					},
				},
			},

			{
				Name:    "check",
				Command: []string{"sh", "./check.sh"},
				Input: cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths: []string{"check.sh"},
						},
					},
				},
			},
		},
	}

	appDir := filepath.Join(r.Dir, appName)

	if err := os.Mkdir(appDir, 0775); err != nil {
		t.Fatal(err)
	}

	if err := app.ToFile(filepath.Join(appDir, baur.AppCfgFile)); err != nil {
		t.Fatalf("writing app cfg file failed: %s", err)
	}

	r.AppCfgs = append(r.AppCfgs, &app)

	buildFilePath := filepath.Join(filepath.Join(appDir, "build.sh"))
	checkFilePath := filepath.Join(filepath.Join(appDir, "check.sh"))

	fstest.WriteToFile(t, []byte(`
#!/bin/sh

echo "building app"
cat output_content.txt > output
`),
		buildFilePath)

	fstest.Chmod(t, buildFilePath, os.ModePerm)

	fstest.WriteToFile(t, []byte("1"), filepath.Join(appDir, "output_content.txt"))

	fstest.WriteToFile(t, []byte(`
#!/bin/sh

echo "check successful"
`),
		checkFilePath)

	fstest.Chmod(t, checkFilePath, os.ModePerm)

	return &app
}

type Repo struct {
	AppCfgs             []*cfg.App
	Dir                 string
	FilecopyArtifactDir string
}

// TaskIDs returns the tasks ids (<AppName>.<TaskName>) of all tasks in the AppCfgs slice
func (r *Repo) TaskIDs() []string {
	var result []string

	for _, appCfg := range r.AppCfgs {
		for _, task := range appCfg.Tasks {
			result = append(result, fmt.Sprintf("%s.%s", appCfg.Name, task.Name))
		}
	}

	return result
}

type repoOptions struct {
	keepTmpDir  bool
	createNewDB bool
}

type Opt func(*repoOptions)

func WithKeepTmpDir() Opt {
	return func(o *repoOptions) {
		o.keepTmpDir = true
	}
}

// WithNewDB create a new database with an unique name and use it for the baur
// repository.
func WithNewDB() Opt {
	return func(o *repoOptions) {
		o.createNewDB = true
	}
}

// CreateBaurRepository creates a new baur repository in a temporary directory
// and a new postgres database with a unique name.
// The funcion changes the current working directory to the created repository directory.
func CreateBaurRepository(t *testing.T, opts ...Opt) *Repo {
	t.Helper()

	var dbURL string
	var options repoOptions

	for _, opt := range opts {
		opt(&options)
	}

	if options.createNewDB {
		var err error

		dbName := dbtest.UniqueDBName()

		t.Logf("creating database %s", dbName)
		if dbURL, err = dbtest.CreateDB(dbName); err != nil {
			t.Fatalf("creating db failed: %s", err)
		}
	} else {
		dbURL = dbtest.PSQLURL()
	}

	t.Logf("database url: %q", dbURL)

	tempDir, err := ioutil.TempDir("", "baur-filesrc-test")
	if err != nil {
		t.Fatal(err)
	}

	if !options.keepTmpDir {
		t.Cleanup(func() { os.RemoveAll(tempDir) })
	}

	artifactDir := filepath.Join(tempDir, "filecopy-artifacts")

	t.Logf("creating baur repository in %s", tempDir)

	if err := os.Mkdir(artifactDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	cfgR := cfg.Repository{
		ConfigVersion: cfg.Version,

		Discover: cfg.Discover{
			Dirs:        []string{"."},
			SearchDepth: 10,
		},

		Database: cfg.Database{
			PGSQLURL: dbURL,
		},
	}

	if err := cfgR.ToFile(filepath.Join(tempDir, baur.RepositoryCfgFile)); err != nil {
		t.Fatalf("could not write repository cfg file: %s", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("changing directory to %q failed: %q", tempDir, err)
	}

	t.Logf("changed working directory to baur repository: %q", tempDir)

	return &Repo{
		Dir:                 tempDir,
		FilecopyArtifactDir: artifactDir,
	}
}
