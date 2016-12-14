#!/bin/ksh

function makeHeader {

d=$1
f=$2

cat <<EOF

package ibmmq

/*
This file was generated from $d/$f using
the hdr.sh script.
*/

var (
EOF

cat $d/$f |\
grep -v "8 byte" |\
grep -v "4 byte" |\
grep -v "CURRENT_LENGTH" |\
awk '
      BEGIN {doprint=0}
      /MQI_BY_NAME_STR/ {
                          doprint=1
                        }

      /{/              {
                          if ((doprint==1) && (match($0,"},$") > 0))
                          {
                            const=$2
                            val=$4
                            if (match($0,"byte") == 0 && match($0,"MQAT_DEFAULT") == 0)
                            {
                              gsub("\"","",const)
                              printf("%-32.32s int32 = %s\n",const,val)
                            }
                          }
                        }

    END { printf(")\n") }
    '
}

d="/Localdev/metaylor/mf/GitHub/ibm-messaging/mq-golang/src/product"
makeHeader $d cmqstrc.h.amd64_linux2 > cmqc_linux.go
makeHeader $d cmqstrc.h.amd64_nt_4   > cmqc_windows.go

go fmt cmqc*.go
