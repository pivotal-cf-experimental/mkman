package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/mkman/config"
	"github.com/cloudfoundry/mkman/manifestgenerator"
	"github.com/cloudfoundry/mkman/stubmakers"
	"github.com/cloudfoundry/mkman/tarball"

	"github.com/cloudfoundry/mkman/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type ManifestGenerator interface {
	GenerateManifest() (string, error)
}

type CreateManifestsCommand struct {
	OutputWriter io.Writer
	ConfigPath   string `long:"config" short:"c" required:"true" description:"Configuration file (required)"`
}

func (command *CreateManifestsCommand) Execute(args []string) error {
	configFileContents, err := ioutil.ReadFile(command.ConfigPath)
	if err != nil {
		return err
	}

	config := config.Config{}
	err = yaml.Unmarshal(configFileContents, &config)
	if err != nil {
		return err
	}

	tarballReader := tarball.NewTarballReader(config.StemcellPath)
	stemcellStubMaker := stubmakers.NewStemcellStubMaker(tarballReader, config.StemcellPath)
	releaseStubMaker := stubmakers.NewReleaseStubMaker(config.CFPath)
	manifestGenerator := manifestgenerator.NewSpiffManifestGenerator(stemcellStubMaker, releaseStubMaker, config.StubPaths, config.CFPath)

	manifest, err := manifestGenerator.GenerateManifest()
	if err != nil {
		return err
	}

	if command.OutputWriter == nil {
		command.OutputWriter = os.Stdout
	}

	_, err = fmt.Fprintf(command.OutputWriter, manifest)
	return err
}
