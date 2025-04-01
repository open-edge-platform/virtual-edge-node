# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

import os
import sys
import yaml
from typing import Dict, Any, List, Tuple

from Rx import Factory
from yaml_schemas import (
    get_ansible_schemas,
    get_default_playbook_schema
)

class YamlValidator:
    def __init__(self, use_fallback: bool = False):
        """
        If use_fallback=True, we will use a default schema for unknown files
        instead of erroring out.
        """
        self.rx_factory = Factory({'register_core_types': True})
        self.schemas = get_ansible_schemas()
        self.use_fallback = use_fallback
        self.fallback_schema = get_default_playbook_schema() if use_fallback else None

    def load_yaml_file(self, file_path: str) -> Tuple[bool, Any]:
        """Load and parse a YAML file."""
        try:
            with open(file_path, 'r') as file:
                content = yaml.safe_load(file)
                return True, content
        except yaml.YAMLError as e:
            return False, f"Error parsing YAML file: {e}"
        except FileNotFoundError:
            return False, f"File not found: {file_path}"
        except Exception as e:
            return False, f"Unexpected error: {e}"

    def validate_yaml(self, content: Any, schema: Dict[str, Any], filename: str) -> Tuple[bool, str]:
        """Validate YAML content against the provided schema."""
        try:
            schema_check = self.rx_factory.make_schema(schema)
            schema_check.check(content)  # Raises an exception on validation failure
            return True, f"✓ {filename} is valid."
        except Exception as e:
            return False, f"✗ {filename} validation error:\n   {str(e)}"

    def validate_directory(self, directory_path: str) -> List[Tuple[str, bool, str]]:
        """
        Validate all known YAML files in a directory.
        If use_fallback=True, unknown files get validated with the fallback schema.
        If use_fallback=False, unknown files produce an error message.
        """
        results = []
        yaml_files = [
            f for f in os.listdir(directory_path)
            if f.endswith('.yml') or f.endswith('.yaml')
        ]

        for filename in yaml_files:
            # Skip secret.yml if desired
            if filename == 'secret.yml':
                continue

            file_path = os.path.join(directory_path, filename)

            # Load the file
            success, content = self.load_yaml_file(file_path)
            if not success:
                # Could not parse or read the file
                results.append((filename, False, content))
                continue

            # Pick a schema
            if filename in self.schemas:
                schema = self.schemas[filename]
                is_valid, message = self.validate_yaml(content, schema, filename)
                results.append((filename, is_valid, message))
            else:
                if self.use_fallback and self.fallback_schema is not None:
                    # Validate as a standard playbook
                    is_valid, message = self.validate_yaml(content, self.fallback_schema, filename)
                    results.append((filename, is_valid, f"[Using fallback schema] {message}"))
                else:
                    # No schema for this file => fail or warn
                    results.append((filename, False, f"No schema defined for {filename}"))

        return results

def main():
    if len(sys.argv) != 2:
        print("Usage: python yaml_validator.py <directory_path>")
        sys.exit(1)

    directory_path = sys.argv[1]
    if not os.path.isdir(directory_path):
        print(f"Error: {directory_path} is not a directory")
        sys.exit(1)

    # Create validator. Set use_fallback=True if you want new .yml files
    # to be treated as normal playbooks by default.
    validator = YamlValidator(use_fallback=False)
    results = validator.validate_directory(directory_path)

    # Print validation results
    print("\nValidation Results:")
    print("-" * 50)

    valid_count = sum(1 for _, is_valid, _ in results if is_valid)
    total_count = len(results)

    for filename, is_valid, message in sorted(results):
        print(f"{message}")

    print("\nSummary:")
    print(f"Valid files: {valid_count}/{total_count}")

    if valid_count != total_count:
        print("\nInvalid files:")
        for filename, is_valid, message in sorted(results):
            if not is_valid:
                print(f"- {filename}")
        sys.exit(1)

    sys.exit(0)

if __name__ == "__main__":
    main()
