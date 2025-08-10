# **Vanity Domain Manager**

## **Overview**

The **Vanity Domain Manager** is a microservice designed to automate the lifecycle of vanity domains. It listens for job requests to add, update, or delete domains, automatically handling the verification of DNS records and the creation of Kubernetes ingress resources.

This service simplifies the process of configuring custom domains for your applications, providing a robust and reliable way to link a user's domain to your Kubernetes services.

## **Deployment**

The project is packaged as a Docker container, geekgonecrazy/vanitydomainmanager. The following Kubernetes manifest files are provided in the examples directory to help you deploy the service to your cluster.

* k8s-configmap.yaml: Defines a ConfigMap that holds the service's configuration, including NATS connection details and system settings.  
* k8s-deployment.yaml: Creates a Kubernetes Deployment that runs the vanityDomainManager container, mounting the configuration from the ConfigMap.  
* k8s-rbac.yaml: Configures the necessary Role, RoleBinding, and ServiceAccount to grant the service permissions to manage ingresses within the Kubernetes cluster.

## **Usage**

You can submit jobs to the Vanity Domain Manager using one of two methods: an HTTP POST request or a NATS message.

### **Method 1: HTTP API**

Submit a POST request to the service's jobs endpoint: service:9595/v1/jobs.

The request body should be a JSON payload with the following structure:

```json
{  
    "referenceId": "bobsyouruncle-com",  
    "type": "add",  
    "domain": {  
        "vanityDomain": "bobsyouruncle3.com",  
        "desiredDnsTargetType": "A",  
        "desiredARecords": [  
            "127.0.0.1"  
        ]  
    }  
}
```

### **Method 2: NATS Messaging**

The service is also a NATS consumer and can process jobs sent to a specific subject.

Publish the same JSON payload as above to the following NATS subject:

{environment}.vanityDomainManager.domainjob.{yourid}

This allows for asynchronous processing and is ideal for systems that are already integrated with NATS.

## **Status and Feedback**

After a job is submitted, the Vanity Domain Manager will publish status updates to a dedicated NATS subject. This enables you to monitor job progress and handle successes or failures without relying on the synchronous nature of a web request.

The status subject is:

{environment}.vanityDomainManager.status.{yourid}

The payload for a status message will be a JSON representation of the following Go struct:

```json
{  
    "success": true,  
    "referenceId": "bobsyouruncle-com",  
    "errorMessage": "",  
    "dropped": false  
}  
```
