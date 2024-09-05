#!/bin/bash

ldapadd -v -d 5 -H ldap://127.0.0.1:8000 -D "cn=Dexter McClary,dc=georgiboy,dc=dev" <<EOF
dn: cn=Dexter McClary,dc=georgiboy,dc=dev
objectClass: person
commonName: Dexter
sn: McClary
EOF
