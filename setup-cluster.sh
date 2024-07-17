talosctl --talosconfig secrets/talosconfig config endpoint \
  `pulumi stack output --json | jq -r '.NetworkInterfaces[0].ip'`

talosctl --talosconfig secrets/talosconfig config node \
  `pulumi stack output --json | jq -r '.NetworkInterfaces[0].ip'`

talosctl --talosconfig secrets/talosconfig bootstrap