#!/bin/sh

rm -f icon.go
cat <<EOF > icon.go
package icon

var Data = []byte{
$(hexdump -ve '"\t" 8/1 "0x%02x, " "\n"' icon.ico | sed -e 's/0x  /0x00/g')
}
EOF
