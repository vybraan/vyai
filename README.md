# vyai
I got tired of using `curl` to make API calls to Gemini just to have an "AI" in my terminal. I wanted something that could also maintain context. The web felt and is bloated, so I built this snazzy CLI tool instead.

## Usage

1. Get a Gemini API key from ai.google.dev.
2. Export the key as an environment variable:
```bash
export GOOGLE_API_KEY=your_api_key_here
```
3. Install the application
### Arch Linux
You can install `vyai` from the AUR using an AUR helper like `yay` or `paru`:
```bash
yay -S vyai
# or
paru -S vyai
```
Else you can install it manually using `git` and `makepkg`:
```bash
git clone https://aur.archlinux.org/vyai.git 
cd vyai
makepkg -si
```

### Manually / From Source
1. Clone the repository:
```bash
3. Run the application using:
```bash
make run
```
or
```bash
go run cmd/main.go
```

### Distro specific packages

While not a comprehensive list, [repology](https://repology.org/project/vyai/versions) provides a decent list of distro
specific packages.

[![Packaging status](https://repology.org/badge/vertical-allrepos/vyai.svg)](https://repology.org/project/vyai/versions)


## Keyboard Shortcuts
- Enter → Send message
- Ctrl + C → Close the app
- Tab / Ctrl + Right → Next tab
- Shift + Tab / Ctrl + Left → Previous tab
- ESC → Normal mode
- I → Insert mode
- Ctrl + N → New Chat
- Ctrl + E → Edit Chat with default editor (falback to vi)
- / → Search in chats
- j/down → scroll down
- k/up → scroll up
- g/home → scroll to start
- G/end → scroll to end
- ? → Show/Close help
## Demo
![vyai](https://raw.githubusercontent.com/vybraan/vyai/refs/heads/master/assets/vyai-demo.gif)
