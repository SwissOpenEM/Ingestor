package metadataextractor

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	b64 "encoding/base64"

	"github.com/google/go-github/github"
	"golift.io/xtractr"
)

type Method struct {
	// Id and display name of the method
	Name string
	// Base64 encoded schema
	Schema string
	// Id and name of the corresponding extractor
	Extractor string
}

type Extractor struct {
	// Path or command to executable
	ExecutablePath string
	// Template of the command line passed to the executable. Will be split into a list of args.
	// It is expected to contain '{{.SourceFolder}}', '{{.OutputFile}}' and if applicable '{{.AdditionalParameters}}'
	templ *template.Template
	// Additional args as string
	AdditionalArgs string
	Version        string
}

type ExtractorInvokationParameters struct {
	Executable           string
	SourceFolder         string
	OutputFile           string
	AdditionalParameters string
}

// Struct to store methods and extractors
type ExtractorHandler struct {
	methods      map[string]Method
	extractors   map[string]Extractor
	outputFolder string
	timeout      time.Duration
}

type ExtractionRequestError struct {
	Message string
}

func (e ExtractionRequestError) Error() string {
	return e.Message
}

// creates a formatted ExtractionRequestError
func reqErrorf(format string, args ...interface{}) ExtractionRequestError {
	return ExtractionRequestError{
		Message: fmt.Sprintf(format, args...),
	}
}

func IsValidJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// Creates a new extractor handler by reading the config files and
// - download missing extractors (optional)
// - verify extractors
// - verify the template command line to invoke the extractor
// - register methods of the extractor in a global map
// - read and validate the schemas associated with the methods
func NewExtractorHandler(config ExtractorsConfig) *ExtractorHandler {
	h := ExtractorHandler{
		outputFolder: path.Join(os.TempDir(), "openem-ingestor", "metadata-extractor"),
		extractors:   map[string]Extractor{},
		methods:      map[string]Method{},
	}

	for _, extractorConfig := range config.Extractors {
		log().Info("Installing Extractor", "name", extractorConfig.Name)

		full_install_path := path.Join(config.InstallationPath, extractorConfig.GithubOrg, extractorConfig.GithubProject, extractorConfig.Version, extractorConfig.Executable)

		if config.DownloadMissingExtractors {
			err := downloadExtractor(full_install_path, extractorConfig)
			if err != nil {
				log().Error("Failed to download extractor", "name", extractorConfig.Name)
				continue
			}
		}

		if err := verifyInstallation(full_install_path, extractorConfig); err != nil {
			log().Error("Installation verification failed", "error", err.Error(), "name", extractorConfig.Name, "path", full_install_path)
			continue
		}

		tmpl, err := template.New(extractorConfig.Name).Parse(extractorConfig.CommandLineTemplate)
		if err != nil {
			log().Error("Failed to parse extractor commandline template", "name", extractorConfig.Name, "template", extractorConfig.CommandLineTemplate)
			continue
		}

		h.timeout = config.Timeout

		for _, m := range extractorConfig.Methods {
			if _, exists := h.methods[m.Name]; exists {
				log().Error("Duplicate method name found. Skipping.", "method", m.Name)
				continue
			}

			schemaPath := path.Join(config.SchemasLocation, m.Schema)

			if config.DownloadSchemas {
				response, err := http.Get(m.Url)
				if err != nil {
					fmt.Println(err)
					log().Error("Failed to download schema for method. Skipping.", "method", m.Name, "url", m.Url)
					continue
				}

				if _, err := os.Stat(config.SchemasLocation); errors.Is(err, os.ErrNotExist) {
					err = os.Mkdir(config.SchemasLocation, os.ModePerm)
					if err != nil {
						log().Error("Failed to create schema directory", "folder", config.SchemasLocation)
						continue
					}
				}

				defer response.Body.Close()
				outFile, err := os.Create(schemaPath)
				if err != nil {
					log().Error("Failed to create schema file for method. Skipping.", "method", m.Name, "url", schemaPath)
					continue
				}
				_, err = io.Copy(outFile, response.Body)
				if err != nil {
					log().Error("Failed to create schema file for method. Skipping.", "method", m.Name, "url", schemaPath)
					continue
				}
			}

			if _, err := os.Stat(schemaPath); errors.Is(err, os.ErrNotExist) {
				log().Error("Schema file not found. Skipping.", "method", m.Name, "file", schemaPath)
				continue
			}

			schema, err := os.ReadFile(schemaPath)
			if err != nil {
				log().Error("Failed to read schema file. Skipping.", "method", m.Name, "file", schemaPath, "error", err.Error())
				continue
			}

			if !IsValidJSON(string(schema)) {
				log().Error("Schema file does not contain valid json. Skipping.", "method", m.Name, "schema", m.Schema)
				continue
			}

			h.methods[m.Name] = Method{
				Name:      m.Name,
				Schema:    b64.StdEncoding.EncodeToString(schema),
				Extractor: extractorConfig.Name,
			}
			log().Debug("Successfully added extractor", "method", m.Name, "extractor", extractorConfig.Name)
		}

		h.extractors[extractorConfig.Name] = Extractor{
			ExecutablePath: full_install_path,
			AdditionalArgs: strings.Join(extractorConfig.AdditionalParameters, " "),
			Version:        extractorConfig.Version,
			templ:          tmpl,
		}
	}

	return &h
}

func verifyInstallation(full_install_path string, extractorConfig ExtractorConfig) error {
	if _, err := os.Stat(full_install_path); errors.Is(err, os.ErrNotExist) {
		return errors.New("expected extractor executable does not exist")
	}
	if _, lookError := exec.LookPath(extractorConfig.Executable); lookError == nil {
		return errors.New("executable file found in PATH of the system")
	}
	return nil
}

func MetadataFilePath(folder string) string {
	hasher := md5.New()
	hasher.Write([]byte(folder))
	hashed_folder := hex.EncodeToString(hasher.Sum(nil))
	return path.Join(os.TempDir(), "openem", "metadata", fmt.Sprintf("%s.json", hashed_folder))
}

func downloadRelease(github_org string, github_proj string, version string, targetFolder string) (string, error) {
	client := github.NewClient(nil)
	opt := &github.ListOptions{Page: 1, PerPage: 10}

	var ctx = context.Background()
	releases, _, err := client.Repositories.ListReleases(ctx, github_org, github_proj, opt)

	if err != nil {
		fmt.Println(err)
	}

	arch := runtime.GOARCH
	if runtime.GOARCH == "amd64" {
		arch = "x86_64"
	}
	OS := runtime.GOOS

	r, _ := regexp.Compile(fmt.Sprintf("(?i)%s_%s_%s", github_proj, OS, arch) + "(\\.tar\\.gz|\\.zip)")

	for _, release := range releases {

		if *release.Name == version {
			for _, asset := range release.Assets {
				if r.MatchString(*asset.Name) {
					url := *asset.BrowserDownloadURL
					fmt.Printf("\n%+v\n", url)
					reader, err := http.Get(url)
					if err != nil {
						log().Error(err.Error())
					}
					defer reader.Body.Close()
					targetFile := path.Join(targetFolder, *asset.Name)
					outFile, _ := os.Create(targetFile)
					defer outFile.Close()

					_, err = io.Copy(outFile, reader.Body)
					if err != nil {
						log().Error(err.Error())
					}
					return outFile.Name(), nil
				}
			}
		}
	}

	return "", nil
}

func verifyFile(file_path string, config ExtractorConfig) (bool, string, error) {

	f, err := os.Open(file_path)
	if err != nil {
		return false, "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return false, "", err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return config.Checksum == checksum, checksum, nil

}

func downloadExtractor(full_install_path string, config ExtractorConfig) error {
	if _, err := os.Stat(full_install_path); errors.Is(err, os.ErrNotExist) {
		targetFolder := os.TempDir()
		file, err := downloadRelease(config.GithubOrg, config.GithubProject, config.Version, targetFolder)
		if err != nil {
			log().Error("error", "error", err.Error())
			return err
		}

		if ok, checksum, err := verifyFile(file, config); err == nil {
			if !ok {
				log().Error("Verification failed", "file", file, "checksum", checksum)
				return errors.New("verification failed")
			} else {
				log().Info("Verification passed", "file", file, "checksum", checksum)
			}
		} else {
			log().Error("Failed to do verification ", "file", file, "error", err.Error())
			return err
		}

		err = os.MkdirAll(path.Dir(full_install_path), 0777)
		if err != nil {
			log().Error("Failed to create folder", "folder", path.Dir(full_install_path))
			return err
		}
		x := &xtractr.XFile{
			FilePath:  path.Clean(file),
			OutputDir: path.Dir(full_install_path),
		}

		size, files, _, err := x.Extract()
		if err != nil || files == nil {
			return fmt.Errorf("Extraction failed %d, %s, %s", size, files, err.Error())
		}
	}
	return nil
}

type MethodAndSchema struct {
	Name   string
	Schema string
}

func (e *ExtractorHandler) AvailableMethods() []MethodAndSchema {
	methods := []MethodAndSchema{}
	if e == nil {
		return methods
	}

	for k, v := range e.methods {
		methods = append(methods, MethodAndSchema{
			k,
			v.Schema,
		})
	}

	sort.SliceStable(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})
	return methods
}

// SplitString split string with a rune comma ignore quoted
func SplitString(str string, r rune) []string {
	quoted := false
	return strings.FieldsFunc(str, func(r1 rune) bool {
		if r1 == '\'' {
			quoted = !quoted
		}
		return !quoted && r1 == r
	})
}

func buildCommandline(templ *template.Template, template_params ExtractorInvokationParameters) (string, []string, error) {
	string_builder := new(strings.Builder)
	err := templ.Execute(string_builder, template_params)
	if err != nil {
		return "", nil, err
	}
	cmdline := strings.TrimSpace(string_builder.String())

	// in order to split cmdline template correctly, quotes are necessary
	args := SplitString(cmdline, ' ')

	// but when passing them to the process, quotes need to trimmed
	for i, arg := range args {
		args[i] = strings.TrimFunc(arg, func(r rune) bool {
			return r == '\''
		})
	}

	// args should be something like ["-i", "/path/to/file1", "-o", "/path/to/file2"]

	binary_path := template_params.Executable
	return binary_path, args, nil
}

type outputCallback func(string)

func runExtractor(ctx context.Context, executable string, args []string, stdout_callback outputCallback, stderr_callback outputCallback) error {
	cmd := exec.CommandContext(ctx, executable, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	var wg sync.WaitGroup
	wg.Add(1)
	go func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			stdout_callback(scanner.Text())
		}
		wg.Done()
	}(bufio.NewScanner(stdout))

	wg.Add(1)
	go func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			stderr_callback(scanner.Text())
		}
		wg.Done()
	}(bufio.NewScanner(stderr))

	err := cmd.Start()

	if err != nil {
		return err
	}

	defer func() {
		err = cmd.Wait()
		wg.Wait()
	}()

	return err
}

func (e *ExtractorHandler) ExtractMetadata(ctx context.Context, method_name string, folder string, output_file string, stdout_callback outputCallback, stderr_callback outputCallback) (string, error) {
	method, ok := e.methods[method_name]

	if !ok {
		return "", reqErrorf("method not found: '%s'", method_name)
	}

	if _, err := os.Stat(folder); err != nil {
		return "", reqErrorf("dataset does not exist")
	}

	extractor, ok := e.extractors[method.Extractor]
	if !ok {
		log().Error("Extractor not found.", "method", method_name)
		return "", fmt.Errorf("extractor not found for the following method: '%s'", method_name)
	}

	err := os.MkdirAll(path.Dir(output_file), 0777)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(output_file); err == nil {
		err := os.Remove(output_file)
		if err != nil {
			return "", err
		}
	}

	params := ExtractorInvokationParameters{
		Executable:           extractor.ExecutablePath,
		SourceFolder:         folder,
		OutputFile:           output_file,
		AdditionalParameters: extractor.AdditionalArgs,
	}

	binary_path, args, err := buildCommandline(extractor.templ, params)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	err = runExtractor(ctx, binary_path, args, stdout_callback, stderr_callback)
	if err != nil {
		return "", err // couldn't run extractor
	}

	if ctx.Err() == context.DeadlineExceeded {
		return "", ctx.Err()
	}

	b, err := os.ReadFile(output_file)
	if err != nil {
		return "", err
	}
	str := string(b)

	if !IsValidJSON(str) {
		return "", errors.New("extractor returned non-valid JSON")
	}
	return str, nil
}
