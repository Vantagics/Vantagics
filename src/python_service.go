package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PythonEnvironment represents a detected Python environment
type PythonEnvironment struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	Type        string `json:"type"` // "System", "Conda", "VirtualEnv"
	IsRecommended bool `json:"isRecommended"`
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

	// 2. Check Standard Conda Locations
	home, _ := os.UserHomeDir()
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

	// 3. Check Common Virtualenv Locations (Optional, e.g., ~/.virtualenvs)
	// You can add more logic here if needed.

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
	err := cmd.Run()
	return err == nil
}

// getPythonVersion runs python --version
func (s *PythonService) getPythonVersion(pythonPath string) string {
	cmd := exec.Command(pythonPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}
