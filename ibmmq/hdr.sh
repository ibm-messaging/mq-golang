#!/bin/ksh

d="/opt/mqm/inc"
f="cmqstrc.h"

(
cat <<EOF

package ibmmq

/*
This file was generated from $d/$f using
the hdr.sh script.
*/

var (
EOF

cat $d/$f |\
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
) > cmqc.go
