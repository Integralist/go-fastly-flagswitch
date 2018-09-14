# go-fastly-flagswitch

```bash
go run main.go --west true
```

Will look for the following `config.json` file:

```json
{
  "services": [
    {
      "name": "your_service_name",
      "id": "your_service_id",
      "dictionary": "your_service_edge_dictionary_id"
    }
  ]
}
```

Then change the relevant edge dictionary's `west` key to have the value `true` or `false`, depending on whether the flag `--west true` was provided.
