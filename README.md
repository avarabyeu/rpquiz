```bash
fly -t ci login -c http://127.0.0.1 -u test -p test
```

```bash
fly -t ci set-pipeline -p rpquiz -c ci/pipeline.yml -l ci/parameters.yml 
```
