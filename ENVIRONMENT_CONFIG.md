# Environment-Based Configuration

## Overview

gojekyll now supports environment-based configuration through an `_admin.yml` file. This allows you to maintain a single configuration file with environment-specific overrides for development, staging, and production environments.

## Usage

### With Environment Overrides

Use the `--env` flag to specify which environment configuration to use:

```bash
gojekyll --env=prod build
gojekyll --env=stg serve
gojekyll --env=dev build --watch
```

### Without Environment Flag

If `_admin.yml` exists and no `--env` flag is provided, gojekyll will automatically use the `base` configuration:

```bash
gojekyll build  # Uses _admin.yml base config if it exists
gojekyll serve  # Uses _admin.yml base config if it exists
```

## Configuration File Structure

Create an `_admin.yml` file in your site's root directory with the following structure:

```yaml
site:
  base:
    # Base configuration that applies to all environments
    title: My Site
    author:
      name: John Doe
      email: john@example.com
    description: My awesome site
    
  prod:
    # Production-specific overrides
    url: https://example.com
    baseurl: ""
    
  stg:
    # Staging-specific overrides
    url: https://staging.example.com
    baseurl: "/staging"
    
  dev:
    # Development-specific overrides
    url: http://localhost:4000
    baseurl: ""
```

## How It Works

1. **Base Configuration**: All environments inherit from the `site.base` section
2. **Environment Overrides**: Values in environment-specific sections (e.g., `site.prod`) override the base values
3. **Deep Merging**: Nested maps are merged recursively, so you only need to specify the values that change
4. **Automatic Loading**: If `_admin.yml` exists, it will be used automatically (with base config) even without the `--env` flag
5. **Fallback**: If `_admin.yml` doesn't exist, gojekyll uses the standard `_config.yml` file

## Example

Given the following `_admin.yml`:

```yaml
site:
  base:
    title: OMG Tennis Foundation
    form_action: https://r2ware.dev/sites/omgtennis/submit
    
  prod:
    envid: prod
    
  stg:
    envid: stg
    form_action: http://rpstg.lan/sites/omgtennis/submit
    
  dev:
    envid: dev
    form_action: http://rpdev.lan:5050/sites/omgtennis/submit
```

Running with different environments:

```bash
# Development environment
gojekyll --env=dev build
# Result: title="OMG Tennis Foundation", form_action="http://rpdev.lan:5050/sites/omgtennis/submit", envid="dev"

# Staging environment
gojekyll --env=stg build
# Result: title="OMG Tennis Foundation", form_action="http://rpstg.lan/sites/omgtennis/submit", envid="stg"

# Production environment
gojekyll --env=prod build
# Result: title="OMG Tennis Foundation", form_action="https://r2ware.dev/sites/omgtennis/submit", envid="prod"
```

## Requirements

- The `_admin.yml` file must have a `site.base` section (required when using environment-based config)
- Environment names can be any string (e.g., "prod", "production", "staging", "dev", etc.)
- If an environment name is specified but doesn't exist in `_admin.yml`, only the base configuration will be used

## Compatibility

- This feature is fully backward compatible
- If no `_admin.yml` exists, gojekyll behaves exactly as before, using `_config.yml`
- Existing sites will continue to work without any changes
- Sites using `_admin.yml` no longer need to specify `--env` to use the base configuration
