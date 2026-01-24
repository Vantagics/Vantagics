@echo off
REM Wrapper for zig c++ to support Go cross-compilation
REM Expects ZIG_EXE to be set in the calling environment

IF "%GOARCH%"=="arm64" (
    set ZIG_ARCH=aarch64
) ELSE (
    set ZIG_ARCH=x86_64
)

"%ZIG_EXE%" c++ -target %ZIG_ARCH%-macos.15.0 %*
