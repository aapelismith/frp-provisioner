#!/bin/sh
set -e -x

if echo ${DEST_IP} | grep -Eq ":"
then
    if [ `cat /proc/sys/net/ipv6/conf/all/forwarding` != 1 ]; then
        exit 1
    fi
    ip6tables -t nat -I PREROUTING ! -s ${DEST_IP}/128 -p ${DEST_PROTO} --dport ${SRC_PORT} -j DNAT --to [${DEST_IP}]:${DEST_PORT}
    ip6tables -t nat -I POSTROUTING -d ${DEST_IP}/128 -p ${DEST_PROTO} -j MASQUERADE
else
    if [ `cat /proc/sys/net/ipv4/ip_forward` != 1 ]; then
        exit 1
    fi
    iptables -t nat -I PREROUTING ! -s ${DEST_IP}/32 -p ${DEST_PROTO} --dport ${SRC_PORT} -j DNAT --to ${DEST_IP}:${DEST_PORT}
    iptables -t nat -I POSTROUTING -d ${DEST_IP}/32 -p ${DEST_PROTO} -j MASQUERADE
fi

sleep 3153600000 # 100 years
