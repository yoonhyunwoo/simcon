* `simcon-oci create`
  * Create container (without running process)
  * Load config.json from bundle directory
  * Mount rootfs
  * Initialize namespaces (PID, NET, etc.) and cgroups
  * Create runtime state directory (/var/lib/simcon/oci/containers/<container-id>/)
