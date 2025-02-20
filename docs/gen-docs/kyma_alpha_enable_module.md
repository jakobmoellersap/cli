---
title: kyma alpha enable module
---

Enables a module in the cluster or in the given Kyma resource.

## Synopsis

Use this command to enable Kyma modules available in the cluster.

### Detailed description

For more information on Kyma modules, see the 'create module' command.

This command enables an available module in the cluster. 
A module is available when it is released with a ModuleTemplate. The ModuleTemplate is used for instantiating the module with proper default configuration.


```bash
kyma alpha enable module [name] [flags]
```

## Examples

```bash

Enable "my-module" from "alpha" channel in "default-kyma" in "kyma-system" Namespace
		kyma alpha enable module my-module -c alpha -n kyma-system -k default-kyma

```

## Flags

```bash
  -c, --channel string     Module's channel to enable.
  -f, --force-conflicts    Forces the patching of Kyma spec modules in case their managed field was edited by a source other than Kyma CLI.
  -k, --kyma-name string   Kyma resource to use. If empty, 'default-kyma' is used. (default "default-kyma")
  -n, --namespace string   Kyma Namespace to use. If empty, the default 'kyma-system' Namespace is used. (default "kyma-system")
  -t, --timeout duration   Maximum time for the operation to enable a module. (default 1m0s)
  -w, --wait               Wait until the given Kyma resource is ready.
```

## Flags inherited from parent commands

```bash
      --ci                  Enables the CI mode to run on CI/CD systems. It avoids any user interaction (such as no dialog prompts) and ensures that logs are formatted properly in log files (such as no spinners for CLI steps).
  -h, --help                Provides command help.
      --kubeconfig string   Path to the kubeconfig file. If undefined, Kyma CLI uses the KUBECONFIG environment variable, or falls back "/$HOME/.kube/config".
      --non-interactive     Enables the non-interactive shell mode (no colorized output, no spinner).
  -v, --verbose             Displays details of actions triggered by the command.
```

## See also

* [kyma alpha enable](kyma_alpha_enable.md)	 - Enables a resource in the Kyma cluster.

