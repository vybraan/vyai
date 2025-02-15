# vyai
I got tired of using `curl` to make API calls to Gemini just to have an "AI" in my terminal. I wanted something that could also maintain context. The web felt and is bloated, so I built this snazzy CLI tool instead.

## Usage

1. Get a Gemini API key from ai.google.dev.
2. Export the key as an environment variable:
```bash
export GOOGLE_API_KEY=your_api_key_here
```
3. Run the application using:
```bash
make run
```
or
```bash
go run cmd/main.go
```

### Keyboard Shortcuts
- Enter → Send message
- Ctrl + C → Close the app
- Tab / Ctrl + Right → Next tab
- Shift + Tab / Ctrl + Left → Previous tab

## Screenshot
![vyai](https://raw.githubusercontent.com/vybraan/vyai/refs/heads/master/assets/screenshot.png)
