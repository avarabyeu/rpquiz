Check Vault installed
```sh
vault operator init -status
```

Check Vault installed
```sh
vault operator init -key-shares=1 -key-threshold=1 | tee -a init_output |
  awk 'BEGIN{OFS=""} /Unseal/ {print "export VAULT_UNSEAL_KEY=",$4};/Root/ {print "export VAULT_ROOT_TOKEN=",$4}' > init_vars

```


```sh
source init_vars
vault operator unseal $VAULT_UNSEAL_KEY
vault auth $VAULT_ROOT_TOKEN 
```

Create a mount in value for use by Concourse pipelines
```sh
vault mount -path=/concourse -description="Secrets for concourse pipelines" generic
```


```sh
vault policy-write policy-concourse concourse_policy.hcl
```

Write secrets to Vault using the following syntax
```sh
vault write concourse/<team-name>/<variable-name> value=<variable-value>
```



quick start:
https://github.com/concourse/concourse-docker/blob/master/docker-compose-quickstart.yml
