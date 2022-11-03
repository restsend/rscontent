Static Markdown Server
=====
Markdown + Django Template server


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
## How to run
```shell
go install github.com/restsend/rscontent@latest

rscontent -r example
```

