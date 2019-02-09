# Peerster with transfers with conditions
## Maxime Dimidschstein

A transfer can be performed by the client through the command line by giving an amount and a destination (who will receive the transferred amount). A condition can also be added, either as a delay or as a date and time. The command may look like one of the following three:
./client -UIPort=10000 -dest=node1 -amount=2
./client -UIPort=10000 -dest=node1 -amount=2 -delay=1d2h3m4s
./client -UIPort=10000 -dest=node1 -amount=2 -date=2019-01-014T23:59:59