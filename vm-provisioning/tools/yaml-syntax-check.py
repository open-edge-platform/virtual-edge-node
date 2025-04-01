# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

import os
import sys
import yaml
import argparse

def is_unquoted_boolean_like(value):
    if isinstance(value, str):
        stripped_value = value.strip()
        is_quoted = (stripped_value.startswith('"') and stripped_value.endswith('"')) or \
                    (stripped_value.startswith("'") and stripped_value.endswith("'"))
        return not is_quoted and stripped_value.lower() in ['true', 'false', 'yes', 'no', 'on', 'off']
    return False

def check_yaml_best_practices(data):
    """
    Check if the YAML data follows best practices.
    """
    issues_found = False
    if isinstance(data, dict):
        for key, value in data.items():
            if isinstance(value, dict):
                if check_yaml_best_practices(value):
                    issues_found = True
            elif isinstance(value, list):
                for item in value:
                    if check_yaml_best_practices(item):
                        issues_found = True
            elif isinstance(value, str):
                if is_unquoted_boolean_like(value):
                    print(f"Warning: The value '{value}' for '{key}' may be interpreted as a boolean. Consider quoting it.")
                    issues_found = True
    elif isinstance(data, list):
        for item in data:
            if check_yaml_best_practices(item):
                issues_found = True
    return issues_found

def test_yaml_file(file_path):
    """
    Test a single YAML file for safe loading and parsing.
    """
    try:
        if os.path.getsize(file_path) > 10 * 1024 * 1024:  # 10 MB limit
            print(f"Warning: The file {file_path} is too large to process safely.")
            return

        with open(file_path, 'r') as stream:
            documents = list(yaml.safe_load_all(stream))
            issues_found = False
            for doc in documents:
                if check_yaml_best_practices(doc):
                    issues_found = True
            if not issues_found:
                print(f"YAML file {file_path} loaded successfully and follows best practices.")
            else:
                print(f"YAML file {file_path} has issues that need to be addressed.")
    except yaml.YAMLError as e:
        print(f"YAML error in file {file_path}: {e}")
    except Exception as e:
        print(f"Unexpected error in file {file_path}: {e}")

def parse_arguments():
    parser = argparse.ArgumentParser(description='Validate YAML files.')
    parser.add_argument('yaml_files', nargs='+', help='List of YAML files to validate')
    parser.add_argument('--ignore', nargs='*', default=[], help='List of directories to ignore')
    return parser.parse_args()

def main():
    args = parse_arguments()
    for file_path in args.yaml_files:
        if not any(file_path.startswith(ignore_dir) for ignore_dir in args.ignore):
            print(f"Testing YAML file: {file_path}")
            test_yaml_file(file_path)

if __name__ == "__main__":
    main()
