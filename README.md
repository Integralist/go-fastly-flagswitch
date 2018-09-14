# go-fastly-flagswitch

Compile a binary and move it to `/usr/local/bin`:

```bash
make build
```

Execute the binary:

```bash
fastly-switch --west true
```

> Note: omit `--west` flag if you want to switch to 'east'

The binary will look for a `config.json` file in the current directory it is running in:

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

This will result in the relevant edge dictionary's `west` key to have the value `true` or `false`, depending on whether the flag `--west true` was provided.

If no `dictionary` key is provided in the `config.json`, then the binary will attempt to create one for you first.

In order to create a new edge dictionary, the binary will first need to locate a non-active service version.

Failing that it'll clone from the latest service version available.

This is because an edge dictionary can only be created when it's associated into a non-active service version.

If this is the case you'll have to manually 'activate' the service yourself (because otherwise fastly will complain that your service includes an edge dictionary that isn't being used -- meaning you'll need to add VCL code to your service in order to _use_ the new edge dictionary).
