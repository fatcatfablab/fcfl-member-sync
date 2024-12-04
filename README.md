# fcfl-member-sync

FatCatFabLab's member syncing facilities.

## How to generate certs
```
step ca certificate --not-after 87600h --offline localhost certs/server.crt certs/server.key
step ca certificate --not-after 87600h --offline memberclient@miquelruiz.net certs/client.crt certs/client.key
```

