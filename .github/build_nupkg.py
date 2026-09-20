#!/usr/bin/env python3
import zipfile, os, sys

version = sys.argv[1]
base = "chocolatey/switchtool"
out = f"dist/switchtool.{version}.nupkg"

content_types = (
    '<?xml version="1.0" encoding="utf-8"?>\n'
    '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">\n'
    '  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>\n'
    '  <Default Extension="nuspec" ContentType="application/octet-stream"/>\n'
    '  <Default Extension="exe" ContentType="application/octet-stream"/>\n'
    '  <Default Extension="txt" ContentType="application/octet-stream"/>\n'
    '</Types>'
)

rels = (
    '<?xml version="1.0" encoding="utf-8"?>\n'
    '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">\n'
    '  <Relationship Type="http://schemas.microsoft.com/packaging/2010/07/manifest"'
    ' Target="/switchtool.nuspec" Id="R1"/>\n'
    '</Relationships>'
)

with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    z.writestr("[Content_Types].xml", content_types)
    z.writestr("_rels/.rels", rels)
    for root, _, files in os.walk(base):
        for f in files:
            full = os.path.join(root, f)
            arcname = os.path.relpath(full, base)
            z.write(full, arcname)

print(f"Created {out}")
