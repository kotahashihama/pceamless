init:
	go get github.com/AlecAivazis/survey/v2
	go get github.com/spf13/cobra
build_bin:
	go build -o build/pceamless ./src/*
