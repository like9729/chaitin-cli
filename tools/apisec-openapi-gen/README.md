# APISec OpenAPI Generator

This tool generates the first static APISec management OpenAPI document from the APISec Django source tree.

Run from the `chaitin-cli` repository root:

```bash
python3 tools/apisec-openapi-gen/generate.py \
  --source /Users/rui.zhu/Documents/workspace/04-开发/tools_llm/product_src/a-vatar/skyview/skyview \
  --output products/apisec/v26.05/openapi.json
```

The generator is intentionally conservative. It scans `views.py` files for exported `*API` classes and HTTP methods, then tries to map `@serialize(SomeSerializer)` decorators to fields in sibling `serializers.py` files. When serializer fields cannot be inferred, the operation is still emitted with `x-cli-body-fallback: true` so the CLI can expose the endpoint through `--body` or `--body-file`.
