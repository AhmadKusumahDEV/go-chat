flow in case for sent message to server have two mehtod 

1. if type message is text will use websocket and waiting job insert to db is not error 
and then publish for run the job push notifiaction to firebase 

2. if client sent a image this will set a type != text , which mean this action will usage endpoint /upload
and the flow will run 3 job 
    1. publish for trigger job and then execute for insert to db 
    2. this job for execute push notifiactoin to firebase 
    3. this job for execute send data from websocket management 

