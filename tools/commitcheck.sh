while read -r line; do
	echo "$line" | jq '.[] | .commit.message' | while read -r message; do
		subject=$(echo -e "$message" | head -n 1)
		body=$(echo -e "$message" | tail -n +2)
		# Check subject line length
		if [ ${#subject} -gt 50 ]; then
			echo "Commit subject exceeds 50 characters: '$subject'"
			exit 1
		fi
		# Check body line length
		echo "$body" | while read -r line; do
			if [ ${#line} -gt 72 ]; then 
				# checks if the line is of the form <1-20 chars>: link/email
				if ! echo "$line" |
						grep -Eq '.{1,20}:\ (http(s)?://)|(.*@.*\..*)' ; then
					echo "Commit body line exceeds 72 characters: '$line'"
					exit 1
				fi
			fi
		done
	done
done
