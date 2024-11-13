package metadataextractor

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/exp/maps"
	"golift.io/xtractr"
)

type Extractor struct {
	ExecutablePath string
	templ          *template.Template
	AdditionalArgs string
	Version        string
}

type ExtractorInvokationParameters struct {
	Executable           string
	SourceFolder         string
	OutputFile           string
	AdditionalParameters string
}

type ExtractorHandler struct {
	extractors   map[string]Extractor
	outputFolder string
}

func NewExtractorHandler(config ExtractorsConfig) *ExtractorHandler {
	h := ExtractorHandler{
		outputFolder: path.Join(os.TempDir(), "openem-ingestor", "metadata-extractor"),
		extractors:   map[string]Extractor{},
	}

	for _, extractorConfig := range config.Extractors {
		slog.Info("Installing Extractor", "name", extractorConfig.Name)

		full_install_path := path.Join(config.InstallationPath, extractorConfig.GithubOrg, extractorConfig.GithubProject, extractorConfig.Version, extractorConfig.Executable)

		if config.DownloadMissingExtractors {
			err := downloadExtractor(full_install_path, extractorConfig)
			if err != nil {
				slog.Error("Failed to download extractor", "name", extractorConfig.Name)
				continue
			}
		}
		if err := verifyInstallation(full_install_path, extractorConfig); err != nil {
			slog.Error("Installation verification failed", "error", err.Error(), "name", extractorConfig.Name, "path", full_install_path)
			continue
		}

		tmpl, err := template.New(extractorConfig.Name).Parse(extractorConfig.CommandLineTemplate)
		if err != nil {
			slog.Error("Failed to parse extractor commandline template", "name", extractorConfig.Name, "template", extractorConfig.CommandLineTemplate)
			continue
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

// install if not exist

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
						slog.Error("error", "error", err.Error())
					}
					defer reader.Body.Close()
					targetFile := path.Join(targetFolder, *asset.Name)
					outFile, _ := os.Create(targetFile)
					defer outFile.Close()

					_, err = io.Copy(outFile, reader.Body)
					if err != nil {
						slog.Error("error", "error", err.Error())
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
			slog.Error("error", "error", err.Error())
			return err
		}

		if ok, checksum, err := verifyFile(file, config); err == nil {
			if !ok {
				slog.Error("Verification failed", "file", file, "checksum", checksum)
				return errors.New("verification failed")
			} else {
				slog.Info("Verification passed", "file", file, "checksum", checksum)
			}
		} else {
			slog.Error("Failed to do verification ", "file", file, "error", err.Error())
			return err
		}

		err = os.MkdirAll(path.Dir(full_install_path), 0777)
		if err != nil {
			slog.Error("Failed to create folder", "folder", path.Dir(full_install_path))
			return err
		}
		x := &xtractr.XFile{
			FilePath:  file,
			OutputDir: path.Dir(full_install_path),
		}

		size, files, _, err := x.Extract()
		if err != nil || files == nil {
			log.Fatal(size, files, err)
		}
	}
	return nil
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
		panic(err)
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

func runExtractor(executable string, args []string, stdout_callback outputCallback, stderr_callback outputCallback) error {
	slog.Info("Executing command", "command", executable, "args", args)

	cmd := exec.Command(executable, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			stdout_callback(scanner.Text())
		}
	}(bufio.NewScanner(stdout))

	go func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			stderr_callback(scanner.Text())
		}
	}(bufio.NewScanner(stderr))

	err := cmd.Start()

	if err != nil {
		slog.Error("Failed to run extractor", "executable", executable, "args", args, "error", err.Error())
		return err
	}

	defer func() { err = cmd.Wait() }()

	return err
}

func (e *ExtractorHandler) ExtractMetadata(extractor_name string, folder string, output_file string, stdout_callback outputCallback, stderr_callback outputCallback) (string, error) {
	if extractor, ok := e.extractors[extractor_name]; ok {

		err := os.MkdirAll(path.Dir(output_file), 0777)
		if err != nil {
			slog.Error("Failed to create folder", "folder", path.Dir(output_file))
			return "", err
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

		if err := runExtractor(binary_path, args, stdout_callback, stderr_callback); err == nil {
			b, err := os.ReadFile(output_file)
			if err != nil {
				slog.Error("Failed to read file", "file", output_file)
				return "", err
			}
			str := string(b)
			fmt.Println(str)
			return str, nil
		}
	slog.Error("Failed to run extractor", "errro", err)
	return "", err
}
