## Quiz Bot

### Configuration

| Parameter    | Default Value              | Description                           |
|-- ---------- | -:-:---------------------- | --:-------------------------------- --|
| PORT         | 4200                       | Server Port                           |
| RP_HOST      | https://rp.epam.com        | ReportPortal URL                      |
| RP_UUID      |                            | ReportPortal UUID                     |
| RP_PROJECT   | andrei_varabyeu_personal   | Project results will be reported to   |
| TG_TOKEN     |                            | Telegram Token                        |
| DB_FILE      | qabot.db                   | Internal Session DB file name         |


### Running in DEV mode (live reloading in enabled)
```sh
docker-compose up --build --force-recreate
```


### Building and running in production mode
```sh
docker-compose -f docker-compose-prod.yml up -d --build --force-recreate
```