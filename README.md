# tiperc20

tiperc20 is a software that behaves as a Slack bot. It handles some form of messages and some kind of reactions, and is able to send arbitrary ERC20 tokens to whom the messages or reactions are sent.

This software is heavily inspired by [tipmona](https://twitter.com/tipmona) and [OKIMOCHI](https://github.com/campfire-inc/OKIMOCHI/).

## Usage

### Register Ethereum Account Address

To receive ERC20 token, you have to register your account address on a certain Ethereum network in advance as below:

```
@tiperc20 register <YOUR_ETH_ACCOUNT_ADDRESS>
```

### Send ERC20 Token

There are 2 ways to send ERC20 token to someone. 

1. `@tiperc20 tip @some_slack_account_name`
2. Add a reaction to someone's message

## Run Your Own tiperc20 Instance

### Settings

tiperc20 requires the environment variables below set:

* `SLACK_BOT_TOKEN`: Slack API Token
* `SLACK_TIP_REACTION`: Reaction name to send a token
* `SLACK_TIP_AMOUNT`: Token amount at one tip
* `ERC20_TOKEN_ADDRESS`: Contract Address of ERC20 token
* `ETH_API_ENDPOINT`: IPC-based RPC endpoint
  * Local Endpoint (ex. `/home/user/.ethereum/testnet/geth.ipc`)
  * Remote API Endpoint (ex. `https://ropsten.infura.io/YOUR_ACCESS_TOKEN`)
* `ETH_KEY_JSON`: JSON string of your account stored in keystore
* `ETH_PASSWORD`: Password for your account on a certain Ethereum network

#### How to Generate Keystore JSON from a Private Key

If you have only a private key, for instance, exported via MetaMask extension for Chrome, you can generate keystore JSON string as below:

```sh
$ geth account import "/path/to/MetaMask kentaro@infura.io Private Key"
```

Then you can find your keystore file under `~/Library/Ethereum/keystore` (if you use macOS):

```sh
$ cat ~/Library/Ethereum/keystore/UTC--XXXXXXXXXXXXXXXX--XXXXXXXXXXXXXXXX | pbcopy
```

Use the string in clipboard as `ETH_KEY_JSON`.

### On Your Local Machine

#### Prerequisites

* PostgreSQL (>= 9.5)
* Go (>= 1.9)
* [dep](https://github.com/golang/dep)

#### Setup

```sh
$ git clone git@github.com:kentaro/tiperc20.git
$ cd tiperc20
$ dep ensure
```

#### DB Migration

```sh
$ export DATABASE_URL "user=postgres dbname=postgres sslmode=disable"
$ go run cmd/goose/main.go postgres $DATABASE_URL up
OK    00001_init.sql
goose: no migrations to run. current version: 1
```

#### Run `tiperc20`

```sh
$ go run main.go token.go -port 20020
```

### On Heroku

#### Create an App

```sh
$ heroku create tiperc20
Creating ⬢ tiperc20... done
https://tiperc20.herokuapp.com/ | https://git.heroku.com/tiperc20.git
```

#### Setup Buildpack and Addon

tiperc20 uses Go.

```sh
$ heroku buildpacks:add heroku/go --app tiperc20
Buildpack added. Next release on tiperc20 will use heroku/go.
Run git push heroku master to create a new release using this buildpack.
```

And it also requires PostgreSQL (>= 9.5).

```sh
$ heroku addons:create heroku-postgresql:hobby-dev --app tiperc20
Creating heroku-postgresql:hobby-dev on ⬢ tiperc20... free
Database has been created and is available
 ! This database is empty. If upgrading, you can transfer
 ! data from another database with pg:copy
Created ********************* as DATABASE_URL
Use heroku addons:docs heroku-postgresql to view documentation
```

#### Set Environment Variables

```sh
$ heroku config:set SLACK_BOT_TOKEN="****************************" --app tiperc20
$ heroku config:set SLACK_TIP_REACTION="tiperc20" --app tiperc20
$ heroku config:set SLACK_TIP_AMOUNT="1000000000000000000" --app tiperc20
$ heroku config:set ERC20_TOKEN_ADDRESS="0x99da589b68821d54721fd7db344bf5e7a4ad3af4" --app tiperc20
$ heroku config:set ETH_API_ENDPOINT="https://ropsten.infura.io/*******************" --app tiperc20
$ heroku config:set ETH_KEY_JSON='{"address": ... }' --app tiperc20
$ heroku config:set ETH_PASSWORD="**********************" --app tiperc20
```

#### Deploy It!

```sh
$ git push heroku master
```

## Author

[Kentaro Kuribayashi](https://kentarok.org)

## License

MIT
