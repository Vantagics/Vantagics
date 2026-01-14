package agent

import (
	"encoding/json"
	"fmt"
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
	requiredPackages := []string{"matplotlib", "numpy", "pandas", "mlxtend", "sqlite3"}
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

// InstallMissingPackages installs the missing packages for a Python environment
func (s *PythonService) InstallMissingPackages(pythonPath string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	// Filter out packages that don't need pip installation
	var installablePackages []string
	var skippedPackages []string
	
	for _, pkg := range packages {
		switch pkg {
		case "sqlite3":
			// sqlite3 is part of Python standard library, skip pip installation
			skippedPackages = append(skippedPackages, pkg)
		default:
			installablePackages = append(installablePackages, pkg)
		}
	}

	// If sqlite3 is missing, it indicates a Python installation issue
	if len(skippedPackages) > 0 {
		for _, pkg := range skippedPackages {
			if pkg == "sqlite3" {
				return fmt.Errorf("sqlite3 is missing from Python standard library. This indicates a problem with your Python installation. Please reinstall Python or use a different Python environment")
			}
		}
	}

	if len(installablePackages) == 0 {
		return nil
	}

	// Check if pip is available
	pipCmd := exec.Command(pythonPath, "-m", "pip", "--version")
	if runtime.GOOS == "windows" {
		pipCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := pipCmd.Run(); err != nil {
		return fmt.Errorf("pip is not available in this Python environment")
	}

	// Install packages one by one to get better error reporting
	for _, pkg := range installablePackages {
		cmd := exec.Command(pythonPath, "-m", "pip", "install", pkg)
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		
		// Set environment variables for better pip behavior
		cmd.Env = append(os.Environ(), 
			"PYTHONIOENCODING=utf-8",
			"PIP_DISABLE_PIP_VERSION_CHECK=1",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to install %s: %v\nOutput: %s", pkg, err, string(output))
		}
	}

	return nil
}

// CreateRapidBIEnvironment creates a dedicated virtual environment for RapidBI
func (s *PythonService) CreateRapidBIEnvironment() (string, error) {
	// Find a suitable base Python interpreter
	basePython, envManager, err := s.findBestPythonForVenv()
	if err != nil {
		return "", fmt.Errorf("no suitable Python interpreter found for creating virtual environment: %v", err)
	}

	// Determine the virtual environment path
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user home directory: %v", err)
	}

	var venvPath string
	var pythonPath string

	switch envManager {
	case "conda":
		// Create conda environment
		venvPath = "rapidbi" // conda env name
		cmd := exec.Command("conda", "create", "-n", "rapidbi", "python", "-y")
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to create conda environment: %v\nOutput: %s", err, string(output))
		}

		// Find the created environment's python path
		pythonPath, err = s.getCondaEnvPythonPath("rapidbi")
		if err != nil {
			return "", fmt.Errorf("failed to locate created conda environment: %v", err)
		}

	case "venv":
		// Create venv environment
		venvPath = filepath.Join(home, ".rapidbi-venv")
		cmd := exec.Command(basePython, "-m", "venv", venvPath)
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to create venv environment: %v\nOutput: %s", err, string(output))
		}

		// Determine python path in the created venv
		if runtime.GOOS == "windows" {
			pythonPath = filepath.Join(venvPath, "Scripts", "python.exe")
		} else {
			pythonPath = filepath.Join(venvPath, "bin", "python")
		}

	default:
		return "", fmt.Errorf("unsupported environment manager: %s", envManager)
	}

	// Install required packages
	requiredPackages := []string{"matplotlib", "numpy", "pandas", "mlxtend"}
	err = s.InstallMissingPackages(pythonPath, requiredPackages)
	if err != nil {
		// Clean up the created environment on failure
		s.cleanupFailedEnvironment(envManager, venvPath)
		return "", fmt.Errorf("failed to install required packages: %v", err)
	}

	return pythonPath, nil
}

// findBestPythonForVenv finds the best Python interpreter for creating virtual environments
// This method reuses the existing ProbePythonEnvironments detection logic for consistency
func (s *PythonService) findBestPythonForVenv() (string, string, error) {
	var detectionErrors []string
	var detectionAttempts []string

	// Method 1: Reuse existing environment detection results
	detectionAttempts = append(detectionAttempts, "Reusing existing Python environment detection...")
	existingEnvs := s.ProbePythonEnvironments()
	
	detectionAttempts = append(detectionAttempts, fmt.Sprintf("Found %d Python environments", len(existingEnvs)))
	
	// Method 1a: First try conda environments (preferred for stability)
	detectionAttempts = append(detectionAttempts, "Checking conda environments...")
	for _, env := range existingEnvs {
		if strings.Contains(strings.ToLower(env.Type), "conda") {
			detectionAttempts = append(detectionAttempts, fmt.Sprintf("Testing conda environment: %s (%s - %s)", env.Path, env.Type, env.Version))
			
			// Verify conda command is available for this environment
			if _, err := exec.LookPath("conda"); err == nil {
				detectionAttempts = append(detectionAttempts, "Conda command available, using conda environment")
				return env.Path, "conda", nil
			} else {
				detectionAttempts = append(detectionAttempts, "Conda command not available, will try venv with this Python")
				// Test if this Python has venv support
				venvCmd := exec.Command(env.Path, "-m", "venv", "--help")
				if runtime.GOOS == "windows" {
					venvCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				}
				
				if err := venvCmd.Run(); err == nil {
					detectionAttempts = append(detectionAttempts, fmt.Sprintf("Conda Python %s has venv support, using venv method", env.Path))
					return env.Path, "venv", nil
				} else {
					detectionErrors = append(detectionErrors, fmt.Sprintf("Conda Python %s lacks venv support: %v", env.Path, err))
				}
			}
		}
	}
	
	// Method 1b: Try system Python environments with venv support
	detectionAttempts = append(detectionAttempts, "Checking system Python environments...")
	for _, env := range existingEnvs {
		if strings.Contains(strings.ToLower(env.Type), "system") {
			detectionAttempts = append(detectionAttempts, fmt.Sprintf("Testing system environment: %s (%s - %s)", env.Path, env.Type, env.Version))
			
			// Test if this Python has venv support
			venvCmd := exec.Command(env.Path, "-m", "venv", "--help")
			if runtime.GOOS == "windows" {
				venvCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}
			
			if err := venvCmd.Run(); err == nil {
				detectionAttempts = append(detectionAttempts, fmt.Sprintf("System Python %s has venv support", env.Path))
				return env.Path, "venv", nil
			} else {
				detectionErrors = append(detectionErrors, fmt.Sprintf("System Python %s lacks venv support: %v", env.Path, err))
			}
		}
	}
	
	// Method 1c: Try any other Python environments with venv support
	detectionAttempts = append(detectionAttempts, "Checking other Python environments...")
	for _, env := range existingEnvs {
		// Skip conda and system as we already checked them
		envTypeLower := strings.ToLower(env.Type)
		if strings.Contains(envTypeLower, "conda") || strings.Contains(envTypeLower, "system") {
			continue
		}
		
		detectionAttempts = append(detectionAttempts, fmt.Sprintf("Testing environment: %s (%s - %s)", env.Path, env.Type, env.Version))
		
		// Test if this Python has venv support
		venvCmd := exec.Command(env.Path, "-m", "venv", "--help")
		if runtime.GOOS == "windows" {
			venvCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		
		if err := venvCmd.Run(); err == nil {
			detectionAttempts = append(detectionAttempts, fmt.Sprintf("Python %s has venv support", env.Path))
			return env.Path, "venv", nil
		} else {
			detectionErrors = append(detectionErrors, fmt.Sprintf("Python %s lacks venv support: %v", env.Path, err))
		}
	}

	// Method 2: Fallback to PATH-based detection (only if no environments found)
	if len(existingEnvs) == 0 {
		detectionAttempts = append(detectionAttempts, "No environments detected, trying PATH-based detection...")
		pythonCandidates := []string{"python3", "python", "py"}
		
		for _, pythonCmd := range pythonCandidates {
			if pythonPath, err := exec.LookPath(pythonCmd); err == nil {
				detectionAttempts = append(detectionAttempts, fmt.Sprintf("Found %s at: %s", pythonCmd, pythonPath))
				
				// Test if venv module is available
				venvCmd := exec.Command(pythonPath, "-m", "venv", "--help")
				if runtime.GOOS == "windows" {
					venvCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				}
				
				if err := venvCmd.Run(); err == nil {
					detectionAttempts = append(detectionAttempts, fmt.Sprintf("%s has venv support", pythonCmd))
					return pythonPath, "venv", nil
				} else {
					detectionErrors = append(detectionErrors, fmt.Sprintf("%s lacks venv support: %v", pythonCmd, err))
				}
			}
		}
	}

	// If we get here, nothing worked - provide comprehensive error message
	errorMsg := "No suitable Python interpreter found for creating virtual environment.\n\n"
	
	// Show what environments were detected
	if len(existingEnvs) > 0 {
		errorMsg += fmt.Sprintf("Detected %d Python environments:\n", len(existingEnvs))
		for i, env := range existingEnvs {
			errorMsg += fmt.Sprintf("%d. %s (%s - %s)\n", i+1, env.Type, env.Version, env.Path)
		}
		errorMsg += "\nHowever, none of them support virtual environment creation.\n\n"
	} else {
		errorMsg += "No Python environments were detected on this system.\n\n"
	}
	
	// Add detection attempts for debugging
	errorMsg += "Detection attempts:\n"
	for _, attempt := range detectionAttempts {
		errorMsg += "• " + attempt + "\n"
	}
	
	if len(detectionErrors) > 0 {
		errorMsg += "\nDetection errors:\n"
		for _, errMsg := range detectionErrors {
			errorMsg += "• " + errMsg + "\n"
		}
	}
	
	errorMsg += "\nTo resolve this issue:\n\n"
	errorMsg += "1. **Install Anaconda/Miniconda (Recommended)**\n"
	errorMsg += "   - Download: https://www.anaconda.com/\n"
	errorMsg += "   - Ensure 'conda' command is available after installation\n\n"
	errorMsg += "2. **Install Python 3.3+ with venv module**\n"
	errorMsg += "   - Download: https://www.python.org/downloads/\n"
	errorMsg += "   - Check 'Add Python to PATH' during installation\n\n"
	errorMsg += "3. **Verify installation**\n"
	errorMsg += "   - Open terminal and try: python --version\n"
	errorMsg += "   - Try: python -m venv --help\n\n"
	errorMsg += "After installation, restart RapidBI and try again."

	return "", "", fmt.Errorf(errorMsg)
}

// getCommonPythonPaths returns common Python installation paths for different platforms
func (s *PythonService) getCommonPythonPaths() []string {
	var paths []string
	
	if runtime.GOOS == "windows" {
		// Windows common paths - expanded to include more versions and locations
		basePaths := []string{
			"C:\\Python37\\python.exe",
			"C:\\Python38\\python.exe",
			"C:\\Python39\\python.exe",
			"C:\\Python310\\python.exe",
			"C:\\Python311\\python.exe",
			"C:\\Python312\\python.exe",
			"C:\\Python313\\python.exe",
			"C:\\Program Files\\Python37\\python.exe",
			"C:\\Program Files\\Python38\\python.exe",
			"C:\\Program Files\\Python39\\python.exe",
			"C:\\Program Files\\Python310\\python.exe",
			"C:\\Program Files\\Python311\\python.exe",
			"C:\\Program Files\\Python312\\python.exe",
			"C:\\Program Files\\Python313\\python.exe",
			"C:\\Program Files (x86)\\Python37\\python.exe",
			"C:\\Program Files (x86)\\Python38\\python.exe",
			"C:\\Program Files (x86)\\Python39\\python.exe",
			"C:\\Program Files (x86)\\Python310\\python.exe",
			"C:\\Program Files (x86)\\Python311\\python.exe",
			"C:\\Program Files (x86)\\Python312\\python.exe",
			"C:\\Program Files (x86)\\Python313\\python.exe",
		}
		paths = append(paths, basePaths...)
		
		// Add user-specific paths
		if home, err := os.UserHomeDir(); err == nil {
			userPaths := []string{
				// Microsoft Store Python installations
				filepath.Join(home, "AppData", "Local", "Microsoft", "WindowsApps", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Microsoft", "WindowsApps", "python3.exe"),
				
				// Standard user installations
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python37", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python38", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python39", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python310", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python311", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python312", "python.exe"),
				filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python313", "python.exe"),
				
				// Roaming installations
				filepath.Join(home, "AppData", "Roaming", "Python", "Python37", "Scripts", "python.exe"),
				filepath.Join(home, "AppData", "Roaming", "Python", "Python38", "Scripts", "python.exe"),
				filepath.Join(home, "AppData", "Roaming", "Python", "Python39", "Scripts", "python.exe"),
				filepath.Join(home, "AppData", "Roaming", "Python", "Python310", "Scripts", "python.exe"),
				filepath.Join(home, "AppData", "Roaming", "Python", "Python311", "Scripts", "python.exe"),
				filepath.Join(home, "AppData", "Roaming", "Python", "Python312", "Scripts", "python.exe"),
				filepath.Join(home, "AppData", "Roaming", "Python", "Python313", "Scripts", "python.exe"),
				
				// Chocolatey installations
				"C:\\ProgramData\\chocolatey\\lib\\python\\tools\\python.exe",
				"C:\\ProgramData\\chocolatey\\lib\\python3\\tools\\python.exe",
				
				// Scoop installations
				filepath.Join(home, "scoop", "apps", "python", "current", "python.exe"),
				filepath.Join(home, "scoop", "shims", "python.exe"),
				filepath.Join(home, "scoop", "shims", "python3.exe"),
			}
			paths = append(paths, userPaths...)
		}
	} else {
		// Unix-like systems (macOS, Linux) - expanded paths
		basePaths := []string{
			"/usr/bin/python3",
			"/usr/bin/python",
			"/usr/local/bin/python3",
			"/usr/local/bin/python",
			"/opt/python/bin/python3",
			"/opt/python/bin/python",
			"/usr/bin/python3.7",
			"/usr/bin/python3.8",
			"/usr/bin/python3.9",
			"/usr/bin/python3.10",
			"/usr/bin/python3.11",
			"/usr/bin/python3.12",
			"/usr/bin/python3.13",
			"/usr/local/bin/python3.7",
			"/usr/local/bin/python3.8",
			"/usr/local/bin/python3.9",
			"/usr/local/bin/python3.10",
			"/usr/local/bin/python3.11",
			"/usr/local/bin/python3.12",
			"/usr/local/bin/python3.13",
		}
		paths = append(paths, basePaths...)
		
		// Add user-specific paths
		if home, err := os.UserHomeDir(); err == nil {
			userPaths := []string{
				// pyenv installations
				filepath.Join(home, ".pyenv", "shims", "python3"),
				filepath.Join(home, ".pyenv", "shims", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.7.0", "bin", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.8.0", "bin", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.9.0", "bin", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.10.0", "bin", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.11.0", "bin", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.12.0", "bin", "python"),
				filepath.Join(home, ".pyenv", "versions", "3.13.0", "bin", "python"),
				
				// Conda installations
				filepath.Join(home, "anaconda3", "bin", "python"),
				filepath.Join(home, "miniconda3", "bin", "python"),
				filepath.Join(home, "miniforge3", "bin", "python"),
				filepath.Join(home, "mambaforge", "bin", "python"),
				
				// Homebrew installations (macOS)
				"/opt/homebrew/bin/python3",
				"/opt/homebrew/bin/python",
				"/usr/local/Cellar/python@3.9/*/bin/python3",
				"/usr/local/Cellar/python@3.10/*/bin/python3",
				"/usr/local/Cellar/python@3.11/*/bin/python3",
				"/usr/local/Cellar/python@3.12/*/bin/python3",
				"/usr/local/Cellar/python@3.13/*/bin/python3",
				
				// Local user installations
				filepath.Join(home, ".local", "bin", "python3"),
				filepath.Join(home, ".local", "bin", "python"),
				filepath.Join(home, "bin", "python3"),
				filepath.Join(home, "bin", "python"),
			}
			paths = append(paths, userPaths...)
		}
		
		// System-wide alternative locations
		altPaths := []string{
			"/opt/anaconda3/bin/python",
			"/opt/miniconda3/bin/python",
			"/usr/local/anaconda3/bin/python",
			"/usr/local/miniconda3/bin/python",
			"/snap/bin/python3",
			"/var/lib/snapd/snap/bin/python3",
		}
		paths = append(paths, altPaths...)
	}
	
	return paths
}

// findCondaBasePython finds the base Python interpreter for conda
// This method is now mainly used as a fallback, as we primarily reuse ProbePythonEnvironments results
func (s *PythonService) findCondaBasePython() string {
	// Method 1: Try to get conda info
	cmd := exec.Command("conda", "info", "--json")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	
	output, err := cmd.Output()
	if err == nil {
		// Parse conda info to get base python
		var condaInfo map[string]interface{}
		if err := json.Unmarshal(output, &condaInfo); err == nil {
			if defaultPrefix, ok := condaInfo["default_prefix"].(string); ok {
				pythonName := "python"
				if runtime.GOOS == "windows" {
					pythonName = "python.exe"
					pythonPath := filepath.Join(defaultPrefix, pythonName)
					if _, err := os.Stat(pythonPath); err == nil {
						return pythonPath
					}
				} else {
					pythonPath := filepath.Join(defaultPrefix, "bin", pythonName)
					if _, err := os.Stat(pythonPath); err == nil {
						return pythonPath
					}
				}
			}
		}
	}

	// Method 2: Reuse existing environment detection (preferred approach)
	envs := s.ProbePythonEnvironments()
	for _, env := range envs {
		if strings.Contains(strings.ToLower(env.Type), "conda") && 
		   (strings.Contains(strings.ToLower(env.Type), "base") || strings.Contains(strings.ToLower(env.Type), "root")) {
			// Test if this python works
			cmd := exec.Command(env.Path, "--version")
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}
			if err := cmd.Run(); err == nil {
				return env.Path
			}
		}
	}

	// Method 3: If no base environment found, use any conda environment
	for _, env := range envs {
		if strings.Contains(strings.ToLower(env.Type), "conda") {
			// Test if this python works
			cmd := exec.Command(env.Path, "--version")
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}
			if err := cmd.Run(); err == nil {
				return env.Path
			}
		}
	}

	return ""
}

// getCondaEnvPythonPath gets the python path for a conda environment
func (s *PythonService) getCondaEnvPythonPath(envName string) (string, error) {
	cmd := exec.Command("conda", "info", "--envs", "--json")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var envInfo map[string]interface{}
	if err := json.Unmarshal(output, &envInfo); err != nil {
		return "", err
	}

	if envs, ok := envInfo["envs"].([]interface{}); ok {
		for _, env := range envs {
			if envPath, ok := env.(string); ok {
				if strings.Contains(envPath, envName) {
					pythonName := "python"
					if runtime.GOOS == "windows" {
						pythonName = "python.exe"
						return filepath.Join(envPath, pythonName), nil
					} else {
						return filepath.Join(envPath, "bin", pythonName), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("conda environment %s not found", envName)
}

// cleanupFailedEnvironment removes a failed environment creation
func (s *PythonService) cleanupFailedEnvironment(envManager, envPath string) {
	switch envManager {
	case "conda":
		cmd := exec.Command("conda", "env", "remove", "-n", envPath, "-y")
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		cmd.Run() // Ignore errors during cleanup
	case "venv":
		os.RemoveAll(envPath) // Remove the directory
	}
}

// CheckRapidBIEnvironmentExists checks if a rapidbi environment already exists
func (s *PythonService) CheckRapidBIEnvironmentExists() bool {
	envs := s.ProbePythonEnvironments()
	for _, env := range envs {
		if strings.Contains(strings.ToLower(env.Type), "rapidbi") {
			return true
		}
	}
	return false
}

// DiagnosePythonInstallation provides detailed diagnostic information about Python installations
func (s *PythonService) DiagnosePythonInstallation() map[string]interface{} {
	result := make(map[string]interface{})
	
	// Check PATH environment
	pathEnv := os.Getenv("PATH")
	result["path_env"] = pathEnv
	
	// Check for conda
	condaInfo := make(map[string]interface{})
	if condaPath, err := exec.LookPath("conda"); err == nil {
		condaInfo["found"] = true
		condaInfo["path"] = condaPath
		
		cmd := exec.Command(condaPath, "--version")
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		if output, err := cmd.CombinedOutput(); err == nil {
			condaInfo["version"] = strings.TrimSpace(string(output))
			condaInfo["working"] = true
		} else {
			condaInfo["working"] = false
			condaInfo["error"] = err.Error()
		}
	} else {
		condaInfo["found"] = false
		condaInfo["error"] = err.Error()
	}
	result["conda"] = condaInfo
	
	// Check for Python commands
	pythonCommands := []string{"python", "python3", "py"}
	pythonInfo := make(map[string]interface{})
	
	for _, cmd := range pythonCommands {
		cmdInfo := make(map[string]interface{})
		if pythonPath, err := exec.LookPath(cmd); err == nil {
			cmdInfo["found"] = true
			cmdInfo["path"] = pythonPath
			
			// Check version
			versionCmd := exec.Command(pythonPath, "--version")
			if runtime.GOOS == "windows" {
				versionCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}
			if output, err := versionCmd.CombinedOutput(); err == nil {
				cmdInfo["version"] = strings.TrimSpace(string(output))
				cmdInfo["working"] = true
				
				// Check venv support
				venvCmd := exec.Command(pythonPath, "-m", "venv", "--help")
				if runtime.GOOS == "windows" {
					venvCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				}
				if err := venvCmd.Run(); err == nil {
					cmdInfo["venv_support"] = true
				} else {
					cmdInfo["venv_support"] = false
					cmdInfo["venv_error"] = err.Error()
				}
			} else {
				cmdInfo["working"] = false
				cmdInfo["error"] = err.Error()
			}
		} else {
			cmdInfo["found"] = false
			cmdInfo["error"] = err.Error()
		}
		pythonInfo[cmd] = cmdInfo
	}
	result["python_commands"] = pythonInfo
	
	// Check common installation paths
	commonPaths := s.getCommonPythonPaths()
	pathsInfo := make([]map[string]interface{}, 0)
	
	for _, path := range commonPaths {
		pathInfo := make(map[string]interface{})
		pathInfo["path"] = path
		
		if _, err := os.Stat(path); err == nil {
			pathInfo["exists"] = true
			
			// Test if it works
			versionCmd := exec.Command(path, "--version")
			if runtime.GOOS == "windows" {
				versionCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			}
			if output, err := versionCmd.CombinedOutput(); err == nil {
				pathInfo["working"] = true
				pathInfo["version"] = strings.TrimSpace(string(output))
			} else {
				pathInfo["working"] = false
				pathInfo["error"] = err.Error()
			}
		} else {
			pathInfo["exists"] = false
		}
		
		pathsInfo = append(pathsInfo, pathInfo)
	}
	result["common_paths"] = pathsInfo
	
	// Check existing environments
	envs := s.ProbePythonEnvironments()
	result["existing_environments"] = envs
	
	// System information
	result["os"] = runtime.GOOS
	result["arch"] = runtime.GOARCH
	
	return result
}
