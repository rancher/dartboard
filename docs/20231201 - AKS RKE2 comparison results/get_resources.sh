#!/bin/sh

NUM=""

get_resource() {
	local num="$1"
	local resource="$2"

	eval $num=$(kubectl get $resource --all-namespaces | wc -l)
	
}


USERS=$(kubectl get users | wc -l)
ROLES=$(kubectl get roles --all-namespaces | wc -l)
ROLEBINDINGS=$(kubectl get rolebindings --all-namespaces | wc -l)
GLBROLES=$()

RESOURCES="users \
	   roles \
	   rolebindings \
	   globalroles \
	   globalrolebindings \
	   configmaps \
	   secrets \
	   projects"

for res in $RESOURCES; do
	get_resource "NUM" "$res"
	echo "$res: $NUM"
done
