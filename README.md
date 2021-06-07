# Lotus Manual Scheduler
### Adding more options to lotus scheduler
<br />

# How come we added manual layout configuration?
Currently, task assignment is automated by the miner scheduler. The scheduler schedules tasks based on the available resources. Once it finds an open window with the required resources, the miner assigns it, and it is not aware of server layouts.

<br /><br />

# When is manual layout configuration necessary?
Consider that we have multiple servers that they do PC1 and PC2 on separate machines. Suppose there are two servers that can both do PC1 or PC2. The data can be transferred between them over the network, while it can also choose a local worker to avoid using the network. This problem only occurs when PC1 + PC2 on more than one machine.


![workflow image](./workflows/MinerPCTasks.jpg)

The PC1 worker will finish its work, report that it finished to the lotus-miner and the
miners' scheduler then looks for a PC2 worker to schedule the next step. The miner is unaware of server layouts. The scheduler looks for workers that support PC2 task type.
Afterwards, the data is transferred between PC1 and PC2.

With the ability to control the flow of a sector, we can select which tasks are carried to which servers. For instance, if we are doing the PC2 job in the same place, no transfer of data is required to server 2.

<br /><br />

# Configuring your layout manually: Getting started
You should set the configuration json file after starting the Lotus project. Set the file in the `$LOTUS_MINER_LAYOUT` directory. 
```sh
export LOTUS_MINER_LAYOUT=/home/user/layout.json
```
The configuration file is in JSON format. If the configuration file is not created or the property `checkLayout` is set to **false**, the scheduler will ignore the layout configuration logic.

The `loadConfigFilePerMinute` option takes a number, and determines how often a config file is updated. For example, if you update a config, it keeps it for this duration

```json
{
  "checkLayout": true,
  "loadConfigFilePerMinute": 1,
  "groups": []
}
```
### This is all you need to set for setting the config file

<br />

| Parameter | Default Value |Description|
| -------------          | ------------- |------------- |
| checkLayout | true  | When set to false, the scheduler ignores the layout configuration logic  |
| loadConfigFilePerMinute           | 1  | Determines how often a config file should be updated  |
| groups           | []  | It includes a list of groups, The group is a list of workers and sectors in one layout  |

<br />

Workers and sectors can be linked as a group. If workers and sectors are on the same group, then we have the concept of **local** server. Base on the environment, we can build a logical server.

By using the `serverName` property, you can assign a name to your logical group. Server names must be unique within each group . Each group contains workers and sectors properties. `workers` property includes list of workers. There is a separate list of `sectors` that specify how other workers are allowed to interact with them.

**There is no difference between a server name and a group name ,We are using server names rather than group names just for clarity**

<br />

## Group structure
```json
{
  "groups": [
    {
      "serverName": "ServerA",
      "workers": [
        {
          "workerId": "worker id"
        }
      ],
      "sectors": [
        {
          "sectorId": "sector id",
          "allowLocalFullControl": true,
          "remoteServers": [
            {
              "serverName": "server name that allowed to interact",
              "allowedTasks": "Supported task types like PC1,PC2"
            }
          ]
        }
      ]
    }
  ]
}
```
| Parameter | Default Value |Description|
| -------------          | ------------- |------------- |
| serverName | required  | It enables you to define the server name, which is required and has to be unique  |
| workers           | []  | List of workers you want to link to the same group, A blank array indicates that your group doesn't have any workers   |
|   worker structure      |   |   |
|    workerId        | required  |  unique worker id created by the miner |
|    *        | *  |  * |
|   sectors      | [] | A list of sector configurations , A blank array indicates that your group doesn't have any sectors  |
|   sector structure      |   |   |
|   sectorId      | required  |  unique sector id created by the miner |
|   allowLocalFullControl      | true  |  When set to true, all workers in the same group have access to the sector on all task types|
|   remoteServers     | [] | Specify how other groups can interact with this sector  |
|   remoteServers.serverName     | required | You can specify which server has access to this sector  |
|   remoteServers.allowedTasks     | required | Depending on the type of task, each server can interact with a sector differently, for example, a server may have access to a sector only for PC1 tasks. It's possible to use * for all types of tasks, or you can use "AP,PC1,PC2" to restrict workers on this server to interact with this sector only based on the tasks that specified.|

<br /><br />

# Using the lotus modified version
main scheduler code. We just added some additional configurations that would let schedulers decide better.
The code can be added manually to your project(no coding skills required) or you can use [forked version](https://github.com/Ja7ad/lotus) . If you want to add the code manually, you have to add the condition to the `sched.go` file.

## Adding the condition     
Find this code in the `sched.go` file.

https://github.com/filecoin-project/lotus/blob/b8deee048eaf850113e8626a73f64b17ba69a9f6/extern/sector-storage/sched.go#L390
```go
if !worker.enabled {
 log.Debugw("skipping disabled worker", "worker", windowRequest.worker)
 continue
}
```
After that, place the following code
```go
if task.taskType != sealtasks.TTFetch && !WorkerHasLayoutAccess(task, windowRequest) {
 continue
}
```

<br />

## Add sched_layout.go file
There is only one file with all the logic implemented. You must place it in the `/extern/sector-storage/` directory.
You can download it from [forked version](https://github.com/Ja7ad/lotus/blob/master/extern/sector-storage/sched_layout.go) or from this repository [this repository](./files/sched_layout.go).
