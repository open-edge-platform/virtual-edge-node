# yaml_schemas.py
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
from typing import Dict, Any

def get_play_schema() -> Dict[str, Any]:
    """
    A base schema for a single Ansible 'play':
      - Must have name, hosts
      - Optionally become, gather_facts, etc.
      - tasks must follow the 'task_schema' below
    """
    return {
        "type": "//rec",
        "required": {
            "name": {"type": "//str"},
            "hosts": {"type": "//str"}
        },
        "optional": {
            "become": {"type": "//bool"},
            "become_user": {"type": "//str"},
            "become_method": {"type": "//str"},
            "gather_facts": {"type": "//bool"},
            "vars_files": {
                "type": "//arr",
                "contents": {"type": "//str"}
            },
            "tasks": {
                "type": "//arr",
                "contents": {"type": "//rec"}  
            }
        }
    }

def get_task_schema() -> Dict[str, Any]:
    """
    A schema that describes a single Ansible task,
    including recognized modules like debug, shell, copy, etc.
    """
    debug_task = {
        "type": "//rec",
        "optional": {
            "msg": {"type": "//str"},
            "var": {"type": "//str"}
        }
    }

    file_task = {
        "type": "//rec",
        "optional": {
            "path": {"type": "//str"},
            "state": {"type": "//str"},
            "mode": {"type": "//str"}
        }
    }

    expect_task = {
        "type": "//rec",
        "optional": {
            "command": {"type": "//str"},
            "responses": {"type": "//map", "values": {"type": "//str"}},
            "timeout": {"type": "//str"}
        }
    }

    copy_task = {
        "type": "//rec",
        "optional": {
            "content": {"type": "//str"},
            "dest": {"type": "//str"},
            "mode": {"type": "//str"},
            "src": {"type": "//str"}
        }
    }

    return {
        "type": "//rec",
        "optional": {
            "name": {"type": "//str"},
            "become": {"type": "//bool"},
            "become_user": {"type": "//str"},
            "become_method": {"type": "//str"},
            "when": {"type": "//any"},
            "register": {"type": "//str"},
            "debug": debug_task,
            "shell": {"type": "//str"},
            "command": {"type": "//str"},
            "copy": copy_task,
            "set_fact": {"type": "//map", "values": {"type": "//any"}},
            "expect": expect_task,
            "fetch": {"type": "//map", "values": {"type": "//any"}},
            "fail": {"type": "//map", "values": {"type": "//any"}},
            "ansible.builtin.file": file_task,
            "ansible.builtin.openssh_keypair": {"type": "//map", "values": {"type": "//any"}},
            "ansible.builtin.apt": {"type": "//map", "values": {"type": "//any"}},
            "ansible.builtin.shell": {"type": "//map", "values": {"type": "//any"}},
            "loop": {"type": "//any"},
            "delegate_to": {"type": "//str"},
            "no_log": {"type": "//bool"},
            "changed_when": {"type": "//bool"},
            "ignore_errors": {"type": "//bool"},
            "async": {"type": "//int"},
            "poll": {"type": "//int"},
            "block": {
                "type": "//arr",
                "contents": {"type": "//map", "values": {"type": "//any"}}
            },
            "rescue": {
                "type": "//arr",
                "contents": {"type": "//map", "values": {"type": "//any"}}
            }
        }
    }

def get_inventory_schema() -> Dict[str, Any]:
    """
    Schema for inventory.yml (top-level 'all->vars->...; all->hosts->...')
    """
    return {
        "type": "//rec",
        "required": {
            "all": {
                "type": "//rec",
                "required": {
                    "vars": {
                        "type": "//rec",
                        "required": {
                            "ansible_vm_deploy_scripts": {"type": "//str"},
                            "ansible_secret_file_path": {"type": "//str"},
                            "ansible_timeout_for_create_vm_script": {"type": "//int"},
                            "ansible_timeout_for_install_vm_dependencies": {"type": "//int"}
                        }
                    },
                    "hosts": {
                        "type": "//map",
                        "values": {
                            "type": "//rec",
                            "optional": {
                                "ansible_host": {"type": "//any"},
                                "ansible_user": {"type": "//any"},
                                "ansible_password": {"type": "//str"},
                                "ansible_become": {"type": "//bool"},
                                "ansible_become_pass": {"type": "//str"},
                                "ansible_become_method": {"type": "//str"},
                                "ansible_become_user": {"type": "//str"},
                                "copy_path": {"type": "//str"},
                                "number_of_vms": {"type": "//int"},
                                "install_packages": {"type": "//int"}
                            }
                        }
                    }
                }
            }
        }
    }

def get_ansible_schemas() -> Dict[str, Any]:
    """
    Return a dict mapping each known file to its schema.
    If you want to handle new files without changing code,
    see 'get_default_playbook_schema()' below for a fallback approach.
    """
    # Build a "task_schema" we can embed in the "play_schema"
    play_schema = get_play_schema()
    task_schema = get_task_schema()

    if "tasks" in play_schema["optional"]:
        play_schema["optional"]["tasks"]["contents"] = task_schema

    array_of_plays = {
        "type": "//arr",
        "contents": play_schema
    }

    return {
        "install_vm_dependencies.yml": array_of_plays,
        "calculate_max_vms.yml": array_of_plays,
        "ssh_key_setup.yml": array_of_plays,
        "show_vms_data.yml": array_of_plays,
        "create_vms.yml": array_of_plays,
        "inventory.yml": get_inventory_schema()
    }

def get_default_playbook_schema() -> Dict[str, Any]:
    """
    OPTIONAL: a fallback schema for any unknown .yml file
    that you still want to treat as a typical Ansible playbook.
    If you define and use this, your script won't fail if new .yml
    files appear without an explicit key.
    """
    play_schema = get_play_schema()
    task_schema = get_task_schema()
    play_schema["optional"]["tasks"]["contents"] = task_schema
    return {
        "type": "//arr",
        "contents": play_schema
    }
