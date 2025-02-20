package kubebuilder

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kyma-project/cli/internal/kustomize"
	"github.com/kyma-project/cli/pkg/step"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/yaml"
)

const (
	V3      = "go.kubebuilder.io/v3"
	V4alpha = "go.kubebuilder.io/v4-alpha"

	projectFile          = "PROJECT"
	configFile           = "config.yaml"
	defaultKustomization = "config/default"
	samplesPath          = "config/samples/"

	crdFileIdentifier = "customresourcedefinition"
	chartsFolder      = "charts/%s"
	templatesFolder   = "templates"
	crdsFolder        = "crds"
)

type Project struct {
	Layout []string `json:"layout,omitempty"`
	Name   string   `json:"projectName,omitempty"`
	Domain string   `json:"domain,omitempty"`
	Repo   string   `json:"repo,omitempty"`
	path   string
}

// ParseProject parses the given kubebuilder project and returns a type containing its metadata and with methods to execute kustomize actions on the project
func ParseProject(path string) (*Project, error) {
	yml, err := os.ReadFile(filepath.Join(path, projectFile))
	if err != nil {
		return nil, err
	}
	p := &Project{}
	if err := yaml.Unmarshal(yml, p); err != nil {
		return nil, fmt.Errorf("could not parse project file: %w", err)
	}

	p.path = path
	return p, nil
}

func (p *Project) FullName() string {
	if p.Domain != "" {
		return fmt.Sprintf("%s/%s", p.Domain, p.Name)
	}
	return p.Name
}

// Build builds the kubebuilder project default kustomization following the given definition.
func (p *Project) Build(name, version string) (string, error) {
	// check layout
	if !(slices.Contains(p.Layout, V3) || slices.Contains(p.Layout, V4alpha)) {
		return "", fmt.Errorf("project layout %v is not supported", p.Layout)
	}

	k, err := kustomize.ParseKustomization(filepath.Join(p.path, defaultKustomization))
	if err != nil {
		return "", err
	}

	// create output folders
	pieces := strings.Split(name, "/")
	chartName := pieces[len(pieces)-1] // always return the last part of the path
	chartsPath := filepath.Join(p.path, fmt.Sprintf(chartsFolder, chartName))
	outPath := filepath.Join(chartsPath, templatesFolder)
	crdsPath := filepath.Join(chartsPath, crdsFolder)

	if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("could not create chart templates output dir: %w", err)
	}
	if err := os.MkdirAll(crdsPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("could not create chart CRDs output dir: %w", err)
	}

	// do build
	yml, err := kustomize.Build(k)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(outPath, "rendered.yaml"), yml, os.ModePerm); err != nil {
		return "", fmt.Errorf("could not write rendered kustomization as yml to %s: %w", outPath, err)
	}

	// move CRDs to their folder
	mvFn := func(path string, d fs.DirEntry, err error) error {
		fileName := filepath.Base(path)
		if strings.Contains(fileName, crdFileIdentifier) {
			if err := os.Rename(path, filepath.Join(crdsPath, fileName)); err != nil {
				return fmt.Errorf("could not move CRD file from %q to %q: %w", path, crdsPath, err)
			}
		}
		return nil
	}

	if err := filepath.WalkDir(outPath, mvFn); err != nil {
		return "", err
	}

	// generate Chart.yaml file
	if err := addChart(chartName, version, chartsPath); err != nil {
		return "", fmt.Errorf("could not generate Chart.yaml file: %w", err)
	}

	return chartsPath, nil
}

func (p *Project) Config() (string, error) {
	configPath := filepath.Join(p.path, configFile)
	info, err := os.Stat(configPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("expected file but found directory at %q", configPath)
	}
	return configPath, nil
}

// DefaultCR checks the samples of the project to obtain the default CR for the operator and returns its contents.
// Should there be several sample files, the user will be asked to specify which one to use.
func (p *Project) DefaultCR(s step.Step) ([]byte, error) {
	// check layout
	if !(slices.Contains(p.Layout, V3) || slices.Contains(p.Layout, V4alpha)) {
		return nil, fmt.Errorf("project layout %v is not supported", p.Layout)
	}

	samplesDir := filepath.Join(p.path, samplesPath)
	filesInDir, err := os.ReadDir(samplesDir)
	if err != nil {
		return nil, fmt.Errorf("could not read samples dir %q: %w", samplesDir, err)
	}

	if len(filesInDir) == 0 {
		return nil, fmt.Errorf("no default CR available: samples directory %q is empty", samplesDir)
	}
	defaultCR := ""
	if len(filesInDir) > 1 {
		// ask for specific file
		var promptString strings.Builder
		promptString.WriteString(fmt.Sprintf("Please specify the file to use as default CR in %s:\n", samplesDir))

		filesMap := map[int]string{}
		fileIndex := 1
		for _, file := range filesInDir {
			if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
				filesMap[fileIndex] = file.Name()
				promptString.WriteString(fmt.Sprintf("[%d] %s\n", fileIndex, file.Name()))
				fileIndex++
			}
		}
		promptString.WriteString(fmt.Sprintln("Press ENTER to select the first option as default."))

		answer, err := s.Prompt(promptString.String())
		if err != nil {
			return nil, fmt.Errorf("could not obtain default CR from user prompt: %w", err)
		}
		var parsedIndex int
		if answer == "" {
			parsedIndex = 1 // Default to the first choice
		} else {
			parsedIndex, err = strconv.Atoi(answer)
			if err != nil {
				return nil, fmt.Errorf("could not obtain default CR from user prompt: %w", err)
			}
		}
		fileName, exists := filesMap[parsedIndex]
		if !exists {
			err = fmt.Errorf("invalid input [%d] for CR selection", parsedIndex)
			return nil, fmt.Errorf("could not obtain default CR from user prompt: %w", err)
		}

		defaultCR = filepath.Join(samplesDir, fileName)
		if err != nil {
			return nil, fmt.Errorf("could not obtain default CR from user prompt: %w", err)
		}
	} else { // use only file in folder
		defaultCR = filepath.Join(samplesDir, filesInDir[0].Name())
	}

	return os.ReadFile(defaultCR)
}
