```bash
fly -t ci login -c http://dev.epm-rpp.projects.epam.com:7000 -u test -p test
```

```bash
fly -t ci set-pipeline -p rpquiz -c ci/pipeline.yml -l ci/parameters.yml 
```