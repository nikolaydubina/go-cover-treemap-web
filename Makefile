build:
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" .
	GOARCH=wasm GOOS=js go build -ldflags="-s -w" -o main.wasm main.go

run: build
	python3 -m http.server 8000

clean:
	-rm wasm_exec.js
	-rm main.wasm