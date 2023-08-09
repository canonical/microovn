#!/usr/bin/env bash

if [ -z $MICROOVN_TEST_ROOT ]; then
    echo "Variable MICROOVN_TEST_ROOT pointing to the root of test folder must be exporter"
    exit 1
fi

function generate_bats() {
    local basename=$1; shift
    local input_file="$TEST_SOURCE$basename"
    local setup_file="$MICROOVN_TEST_ROOT/test_helper/setup_teardown/$basename"
    local output_file="$MICROOVN_TEST_ROOT/${basename//.bash/.bats}"
    echo "processing '$input_file' into '$output_file'"
    # Generate BATS tests from functions defined in the source file
    (
        # shellcheck disable=SC1090 # No need to process dynamically loaded files
        source "$input_file"
        printf "# This is a bash shell fragment -*- bash -*- \n\n" > "$output_file"
        printf "load %s\n" "$setup_file" >> "$output_file"
        printf "load %s\n\n" "$input_file" >> "$output_file"

        while read -r declaration ; do
            local function_name=""
            function_name=$(echo "$declaration" | awk '{print$3;}')
            if [ "$function_name" == "${FUNCNAME[0]}" ]; then
                continue
            fi
            printf "@test \"%s\" {\n" "$function_name" >> "$output_file"
            printf "    %s \n}\n\n" "$function_name" >> "$output_file"
        done < <(declare -F)
    )

    # Process any include statements
    while read -r include_file ; do
        full_include_path="$TEST_SOURCE$include_file"
        # shellcheck disable=SC1090 # No need to process dynamically loaded files
        source "$full_include_path"

        printf "load %s\n\n" "$full_include_path" >> "$output_file"
        while read -r declaration ; do
            local function_name=""
            function_name=$(echo "$declaration" | awk '{print$3;}')
            if [ "$function_name" == "${FUNCNAME[0]}" ]; then
                continue
            fi
            printf "@test \"%s\" {\n" "$function_name" >> "$output_file"
            printf "    %s \n}\n\n" "$function_name" >> "$output_file"
        done < <(declare -F)

    done < <(grep -E "^# include: " "$input_file" | awk '{print $NF;}')


}

TEST_SOURCE="$MICROOVN_TEST_ROOT"/test_helper/src/
SOURCES="$(ls "$TEST_SOURCE")"

for source in $SOURCES ; do
    generate_bats "$source"
done
