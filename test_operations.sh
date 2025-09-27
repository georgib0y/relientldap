#!/usr/bin/env bash

die() {
    echo 1>&2
    echo 1>&2
    echo $1 1>&2
    exit 1
}

LDAP_ADDR="ldap://127.0.0.1:8000"
BIND_DN="cn=Test1,dc=georgiboy,dc=dev"
BIND_PW="password123"

ldapadd -vvv -d 5 -H "$LDAP_ADDR" -D "$BIND_DN" -w "$BIND_PW" <<EOF || die "could not add op"
dn: cn=Dexter McClary,dc=georgiboy,dc=dev
objectClass: person
cn: Dexter
sn: McClary
EOF

echo
echo "--- added dexter ---"
echo


ldapmodify -vvv -d 5 -H "$LDAP_ADDR" -D "$BIND_DN" -w "$BIND_PW" <<EOF || die "could not mod (add) op"
dn: cn=Dexter McClary,dc=georgiboy,dc=dev
changetype: modify
add: description
description: Test Account
-
add: telephoneNumber
telephoneNumber: +61 491 760 129
-
add: userPassword
userPassword: password123
-
EOF

echo
echo "--- added (modified) extra fields to dexter ---"
echo

ldapmodify -vvv -d 5 -H "$LDAP_ADDR" -D "$BIND_DN" -w "$BIND_PW" <<EOF || die "could not mod op"
dn: cn=Dexter McClary,dc=georgiboy,dc=dev
changetype: modify
delete: telephoneNumber
-
replace: userPassword
userPassword: password12345
-
EOF

echo
echo "--- did some more modifications to dexter ---"
echo

