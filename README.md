# Markdown + Django Template Static Generator

Tiny tool for static content.

## Directory layout
```shell

example/
├── config.json
├── content
│   ├── about.md
│   └── index.md
├── static
└── template
    ├── index.html
    └── page.html
```

## How to run dev mode
```shell
go install github.com/restsend/rscontent@latest

# run dev mode
rscontent -r example
```
## Build static sites
All contents build with html, the result store in `./dist` directory

```shell
pi@DESKTOP-SSDOA19:~/workspace/rs/rscontent$ rscontent -r example -b

2022/11/05 11:44:45 static dir: example/static
2022/11/05 11:44:45 content dir: example/content
2022/11/05 11:44:45 template dir: example/template
2022/11/05 11:44:45 
2022/11/05 11:44:45 Build 'example/content'....
2022/11/05 11:44:45  'example/content/about.md' => 'dist/about.html' size 155 B  usage 1 ms
2022/11/05 11:44:45  'example/content/blog/hello.md' => 'dist/blog/hello.html' size 91 B  usage 0 ms
2022/11/05 11:44:45  'example/content/index.md' => 'dist/index.html' size 128 B  usage 0 ms
```
