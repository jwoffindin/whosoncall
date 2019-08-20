# Who's on call?

Authenticated API that returns current on-call person for given schedule.

    curl -H 'x-api-key: secret' https://endpoint/dev/whos-on-call\?schedule\=P538IZH

Will return a JSON object like:

    {"name":"Audrey Satterfield","email":"audrey.satterfield@example.com"}


if there's a problem (e.g. incorrect schedule) you'll get smoething like:

    {"errorMessage":"Unable to determine on-call person","errorType":"errorString"}


## Deploying

Requires golang 1.11+ build environment.

    assume-role orionhealth
    make deploy
