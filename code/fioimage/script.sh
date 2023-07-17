#!/bin/sh
echo "Iniciando o teste de iops"

output=$(fio --randrepeat=1 --ioengine=libaio --direct=1 --gtod_reduce=1 --name=test --filename=test --bs=4k --iodepth=64 --size=1G --readwrite=randrw --rwmixread=75 --output-format=json+ | jq -r '.jobs[0] | .read.iops, .write.iops')

read_iops=$(echo "$output" | awk 'NR==1{ printf("%.0f\n", $1) }')
write_iops=$(echo "$output" | awk 'NR==2{ printf("%.0f\n", $1) }')

echo "Resultado de IOPS de Leitura: $read_iops"
echo "Resultado de IOPS de Escrita: $write_iops"
echo "-----------------------------"
echo "Threshold de IOPS de Leitura: $readThreshold"
echo "Threshold de IOPS de Escrita: $writeThreshold"

if [ "$read_iops" -gt "$readThreshold" ]; then
    echo "O iops de leitura é maior do que o definido no Tester"
else
    echo "O iops de leitura é menor do que o definido no Tester, desativando o Scheduling no node."
    exit 1
fi

if [ "$write_iops" -gt "$writeThreshold" ]; then
    echo "O iops de escrita é maior do que o definido no Tester"
else
    echo "O iops de escrita é menor do que o definido no Tester, desativando o Scheduling no node."
    exit 1
fi

exit 0