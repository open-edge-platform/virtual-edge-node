# How to Add More Files in the Future

To add a new YAML file for validation `new_file.yml` with the schema similar to array_of_plays
add this line to the yaml_validator.py file.

```python
"new_file.yml": array_of_plays,
```

For inventory files or other specific YAML structures, you can create a custom schema by defining a function like:

```python
def get_special_schema():
    # Define custom schema logic
    return custom_schema
```

and use this schema instead of array_of_plays.
