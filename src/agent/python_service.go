package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// PythonEnvironment represents a detected Python environment
type PythonEnvironment struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	Type        string `json:"type"` // "System", "Conda", "VirtualEnv"
	IsRecommended bool `json:"isRecommended"`
}

// PythonExecutor defines the interface for running python scripts
type PythonExecutor interface {
	ExecuteScript(pythonPath string, script string) (string, error)
}

// PythonService handles Python environment detection and validation
type PythonService struct{}

// NewPythonService creates a new PythonService
func NewPythonService() *PythonService {
	return &PythonService{}
}

// ProbePythonEnvironments searches for Python environments
func (s *PythonService) ProbePythonEnvironments() []PythonEnvironment {
	var envs []PythonEnvironment
	seen := make(map[string]bool)

	// Helper to add unique paths
	addEnv := func(path, envType string) {
		if path == "" {
			return
		}
		// Resolve symlinks
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			realPath = path
		}
		
		if _, exists := seen[realPath]; !exists {
			if _, err := os.Stat(realPath); err == nil {
				version := s.getPythonVersion(realPath)
				envs = append(envs, PythonEnvironment{
					Path:    realPath,
					Version: version,
					Type:    envType,
				})
				seen[realPath] = true
			}
		}
	}

	// 1. Check System PATH
	if path, err := exec.LookPath("python3"); err == nil {
		addEnv(path, "System")
	}
	if path, err := exec.LookPath("python"); err == nil {
		addEnv(path, "System")
	}

	// 2. Check Conda environments.txt (Highly reliable for Conda)
	home, _ := os.UserHomeDir()
	condaEnvsFile := filepath.Join(home, ".conda", "environments.txt")
	if data, err := os.ReadFile(condaEnvsFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			envPath := strings.TrimSpace(line)
			if envPath == "" {
				continue
			}
			
			pythonName := "python"
			if runtime.GOOS == "windows" {
				pythonName = "python.exe"
			}

			// Try path/python.exe (Windows) or path/bin/python (Unix)
			fullPath := filepath.Join(envPath, "bin", pythonName)
			if runtime.GOOS == "windows" {
				fullPath = filepath.Join(envPath, pythonName)
			}
			
			envName := filepath.Base(envPath)
			addEnv(fullPath, "Conda ("+envName+")")
		}
	}

	// 3. Check Standard Conda Locations (Fallback/Supplementary)
	condaLocations := []string{
		filepath.Join(home, "anaconda3"),
		filepath.Join(home, "miniconda3"),
		filepath.Join(home, "opt", "anaconda3"),
		filepath.Join(home, "opt", "miniconda3"),
		"/opt/anaconda3",
		"/opt/miniconda3",
	}

	if runtime.GOOS == "windows" {
		condaLocations = append(condaLocations, 
			filepath.Join(home, "Anaconda3"),
			filepath.Join(home, "Miniconda3"),
			`C:\ProgramData\Anaconda3`,
			`C:\ProgramData\Miniconda3`,
			filepath.Join(os.Getenv("APPDATA"), "Local", "Continuum", "anaconda3"),
		)
	}

	for _, base := range condaLocations {
		// Check base env
		pythonName := "python"
		if runtime.GOOS == "windows" {
			pythonName = "python.exe"
		}
		
		basePython := filepath.Join(base, "bin", pythonName)
		if runtime.GOOS == "windows" {
			basePython = filepath.Join(base, pythonName)
		}
		addEnv(basePython, "Conda (Base)")

		// Check envs folder
		envsDir := filepath.Join(base, "envs")
		entries, err := os.ReadDir(envsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					envPython := filepath.Join(envsDir, entry.Name(), "bin", pythonName)
					if runtime.GOOS == "windows" {
						envPython = filepath.Join(envsDir, entry.Name(), pythonName)
					}
					addEnv(envPython, "Conda ("+entry.Name()+")")
				}
			}
		}
	}

	return envs
}

// PythonValidationResult holds the validation check results
type PythonValidationResult struct {
	Valid           bool     `json:"valid"`
	Version         string   `json:"version"`
	MissingPackages []string `json:"missingPackages"`
	Error           string   `json:"error"`
}

// ValidatePythonEnvironment checks the environment for version and packages
func (s *PythonService) ValidatePythonEnvironment(path string) PythonValidationResult {
	result := PythonValidationResult{
		Valid: true,
	}

	// 1. Check existence and version
	if _, err := os.Stat(path); err != nil {
		result.Valid = false
		result.Error = "Python executable not found"
		return result
	}

	result.Version = s.getPythonVersion(path)
	if result.Version == "Unknown" {
		result.Valid = false
		result.Error = "Could not verify Python version"
		return result
	}

	// 2. Check for required packages
	requiredPackages := []string{"pandas", "matplotlib"}
	for _, pkg := range requiredPackages {
		if !s.checkPackage(path, pkg) {
			result.MissingPackages = append(result.MissingPackages, pkg)
		}
	}

	return result
}

func (s *PythonService) checkPackage(pythonPath, packageName string) bool {
	// Run python -c "import package"
	cmd := exec.Command(pythonPath, "-c", "import "+packageName)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	err := cmd.Run()
	return err == nil
}

// getPythonVersion runs python --version
func (s *PythonService) getPythonVersion(pythonPath string) string {
	cmd := exec.Command(pythonPath, "--version")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}

// ExecuteScript runs a Python script and returns the output
func (s *PythonService) ExecuteScript(pythonPath string, script string) (string, error) {
	// Create a temp file for the script
	tmpFile, err := os.CreateTemp("", "rapidbi_script_*.py")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	cmd := exec.Command(pythonPath, tmpFile.Name())
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	// Set UTF-8 encoding for Python output to handle Unicode characters
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
