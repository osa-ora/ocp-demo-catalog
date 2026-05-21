# ocp-demo-catalog
Catalog of OCP demos for the Demo Operator

In this demo, we built an operator who consumes index.ymal file which contains the list of demos available and there metadata, and if the user request a demo, it generates the demo artifacts.

Once, installed as per the file: basitc-operator/definition.yaml

The operator will be available in the software catalog in OpenShift GUI:

<img width="456" height="596" alt="Screenshot 2026-05-21 at 2 39 23 PM" src="https://github.com/user-attachments/assets/93b462e8-41b6-43c6-ab27-1df97221c813" />

Then select it and install it:

<img width="360" height="535" alt="Screenshot 2026-05-21 at 2 39 35 PM" src="https://github.com/user-attachments/assets/d151c844-0530-4070-989a-d960f2384a66" />

<img width="874" height="569" alt="Screenshot 2026-05-21 at 2 48 34 PM" src="https://github.com/user-attachments/assets/aec347cd-d734-4e98-9071-959f3fb3ab96" />

<img width="543" height="253" alt="Screenshot 2026-05-21 at 2 49 29 PM" src="https://github.com/user-attachments/assets/a57576c2-9c04-4e22-8ff3-43e0efa93bda" />

Then you can go and create the demo requests:

<img width="1253" height="665" alt="Screenshot 2026-05-21 at 2 49 39 PM" src="https://github.com/user-attachments/assets/693bfbbc-bcfe-4559-ae23-c0f9c3fa243b" />

<img width="1025" height="608" alt="Screenshot 2026-05-21 at 2 50 08 PM" src="https://github.com/user-attachments/assets/938a2d0f-d6f0-47d8-85e1-3a56ff320ea1" />

And now all demo request resources are related to that request and will be deleted once this demo request is deleted.

<img width="923" height="416" alt="Screenshot 2026-05-21 at 2 50 30 PM" src="https://github.com/user-attachments/assets/ac296fcf-9288-4562-8723-f91a6796e726" />





