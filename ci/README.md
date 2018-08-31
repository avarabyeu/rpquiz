#### 1. Deploy vault
```sh
docker-compose up -d vault
```

#### 2. Make sure Vault CLI is installed. Check connection
```sh
export VAULT_ADDR=https://$DOCKER_IP:8200
export VAULT_SKIP_VERIFY=true
vault operator init -status
```

#### 3. Init Vault
```sh
vault operator init -key-shares=1 -key-threshold=1 | tee -a init_output |
  awk 'BEGIN{OFS=""} /Unseal/ {print "export VAULT_UNSEAL_KEY=",$4};/Root/ {print "export VAULT_ROOT_TOKEN=",$4}' > init_vars

```

#### 4. Configure Vault CLI, Unseal and authenticate
```sh
source init_vars
vault operator unseal $VAULT_UNSEAL_KEY
vault auth $VAULT_ROOT_TOKEN 
```

#### 5. Create a mount in value for use by Concourse pipelines
```sh
vault secrets enable -path=/concourse -description="Secrets for concourse pipelines" generic
```

#### 6. Create policy for concourse
```sh
vault policy write policy-concourse concourse_policy.hcl
```

#### 7. Create a token for concourse and put it to docker-compose
```sh
vault token-create --policy=policy-concourse -period="600h" -format=json
```


#### 7. Deploy concourse
```sh
docker-compose up -d concourse
```

#### 8. Enjoy! Write secrets to Vault using the following syntax
```sh
vault write concourse/<team-name>/<variable-name> value=<variable-value>
```
```sh
vault write concourse/<team-name>/<variable-name> value=<variable-value>
```
```sh
cat myfile.txt | vault write concourse/main/repo_key value=-
vault write concourse/main/docker_host value=my_host:2375
```

vault write concourse/main/repo_key value=<variable-value>

quick start:
https://github.com/concourse/concourse-docker/blob/master/docker-compose-quickstart.yml


Using fly:
```bash
fly -t ci login -c http://127.0.0.1 -u test -p test
```

```bash
fly -t ci set-pipeline -p rpquiz -c ci/pipeline.yml -l ci/parameters.yml 
```
