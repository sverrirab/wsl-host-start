# Manual Installation

If you prefer not to use Scoop, you can install wstart manually from a GitHub release or from source.

## From a release

1. Download the latest `wstart_*_bundle.zip` from [GitHub Releases](https://github.com/sverrirab/wsl-host-start/releases). This contains both the Windows host and WSL binaries.

2. Extract the zip and run the installer from **PowerShell**:

   ```powershell
   .\wstart-host.exe --install
   ```

   This will request **administrator privileges** (UAC prompt) and then:
   - Copy both binaries to `C:\Program Files\wstart\`
   - Create default `config.toml` and `allowlist.toml` (commented out)
   - Generate a signing key and sign the config files

3. In your **WSL session**, create a symlink. The installer prints the exact command for your system, but it will look like:

   ```bash
   mkdir -p ~/.local/bin
   ln -sf "/mnt/c/Program Files/wstart/wstart" ~/.local/bin/wstart
   ```

   Use the path printed by `--install` — it accounts for non-standard Windows drive letters.

4. Ensure `~/.local/bin` is in your PATH. If not, add to `~/.bashrc` or `~/.zshrc`:

   ```bash
   export PATH="$HOME/.local/bin:$PATH"
   ```

5. Test it:

   ```bash
   wstart .
   ```

## From source

Build on any machine (macOS, Linux, Windows with Go 1.24+):

```bash
git clone https://github.com/sverrirab/wsl-host-start.git
cd wsl-host-start
make build
```

This cross-compiles both `bin/wstart` (linux/amd64) and `bin/wstart-host.exe` (windows/amd64). Then run the installer:

```powershell
.\bin\wstart-host.exe --install
```

Follow the WSL symlink instructions printed by the installer.

## Upgrading

Download the new release zip (or `make build`), then run `--install` again — it will overwrite the binaries and re-sign config files. Your existing config and allowlist are preserved.

## Prerequisites

- WSL (1 or 2) with [interop enabled](https://learn.microsoft.com/en-us/windows/wsl/wsl-config#interop-settings) (the default)
- Go 1.24+ (only needed when building from source)
