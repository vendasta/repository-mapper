#!/bin/bash
contributors=$(git shortlog --summary --numbered --email | cut -f2)
numContributors=$(echo contributors | wc -l)

jq -n --argjson "numContributors" "$numContributors" --arg "contributors" "$contributors" '{numContributors: $numContributors, contributors: $contributors | split("\n")}' >&3
