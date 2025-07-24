#!/bin/bash

ldapadd -vvv -d 5 -H ldap://127.0.0.1:8000 -D "cn=Test1,dc=georgiboy,dc=dev" -w password123 <<EOF
dn: cn=Dexter McClary,dc=georgiboy,dc=dev
objectClass: person
cn: Dexter
sn: McClary
EOF
