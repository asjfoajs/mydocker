package nsenter

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

//这里的__attribute__((constructor))指的是，一旦这个包被引用，那么这个函数就会被自动执行
//类似于构造函数，会在程序一启动的时候运行
__attribute__((constructor)) void enter_namespace(void) {
  //这里的代码会在Go运行时启动前执行，它会在单线程的C上下文中运行
  char *mydocker_pid;
  //从环境变量中获取需要进入的PID
  mydocker_pid = getenv("mydocker_pid");
  if(mydocker_pid){
    // fprintf(stdout,"got mydocker_pid=%s\n",mydocker_pid);
  }else {
    //fprintf(stdout,"missing mydocker_pid env skip nsenter");
    //这里，如果没有指定PID,就不需要向下执行，直接退出
    return;
  }

  char *mydocker_cmd;
  //从环境变量里面获取需要执行的命令
  mydocker_cmd = getenv("mydocker_cmd");
  if (mydocker_cmd) {
    // fprintf(stdout,"got mydocker_cmd=%s\n",mydocker_cmd);
  }else {
    fprintf(stdout,"missing mydocker_cmd env skip nsenter");
    //如果没有指定命令，则直接退出
    return;
  }
  int i;
  char nspath[1024];
  //需要进入的五种Namespace
  char *namespaces[] = {"ipc","uts","net","pid","mnt"};

  for (i = 0;i<5;i++) {
    //凭借对应的路径/proc/pid/ns/ipc,类似这样
    sprintf(nspath,"/proc/%s/ns/%s",mydocker_pid,namespaces[i]);
    int fd = open(nspath,O_RDONLY);
    //这里才真正调用setns系统 调用进入对应的Namespace
    if (setns(fd,0) == -1) {
      //fprintf(stderr,"setns on %s namespace failed :%s\n",namespaces[i],stderr(errno));
    }else {
      // fprintf(stdout,"setns on %s namespace succeded\n",namespaces[i]);
    }
    close(fd);
  }
  //在进入的Namespace中执行指定的命令
  int res = system(mydocker_cmd);
  //退出
  exit(0);
  return;
}
*/
import "C"
