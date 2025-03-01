# riq documentation

Viewable at https://libops.github.io/riq/

## Local docs development

```
docker build -t docs:main .
docker run -p 8080:80 docs:main
```

You should be able to view the docs in your web browser at http://localhost:8080
