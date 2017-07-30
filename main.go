package main

import (
	"fmt"
	"os"
	"os/signal"
	"unsafe"
	"syscall"
)

/*
#cgo CFLAGS: -I ./libzt/include
#cgo darwin LDFLAGS: -L ./libzt/darwin/ -lzt -lstdc++

#include "libzt.h"
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <string.h>
#include <netdb.h>
*/
import "C"

const NETWORK_ID = "8056c2e21c000001"
const PORT = 50718 // 7878
const BUF_SIZE = 2000

func setupCleanUpOnInterrupt() chan bool {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	cleanupDone := make(chan bool)

	go func() {
		for range signalChan {
			fmt.Println("\nReceived an interrupt, shutting dow.\n")

			cleanupDone <- true
		}
	}()
	return cleanupDone
}

func getOtherIP() string {
	if len(os.Args) >= 2 {
		return os.Args[1]
	} else {
		return ""
	}
}

func validate(value C.int, message string) {
	if value < 0 {
		fmt.Println(message)
		os.Exit(1)
	}
}

func onError(err error, message string) {
	if err != nil {
		fmt.Printf("%s: %v", message, err)
		os.Exit(1)
	}
}

func bindAndListen(sockfd C.int) int {
	//serverSocket := C.struct_sockaddr_in6{sin6_flowinfo: 0, sin6_family: C.AF_INET6, sin6_addr: C.in6addr_any, sin6_port: 7878}
	serverSocket := syscall.RawSockaddrInet6{Flowinfo: 0, Family: syscall.AF_INET6, Port: PORT}
	retVal := C.zts_bind(sockfd, (* C.struct_sockaddr)(unsafe.Pointer(&serverSocket)), C.sizeof_struct_sockaddr_in6)
	validate(retVal, "ERROR on binding")
	fmt.Println("Bind Complete")

	C.zts_listen(sockfd, 1)
	fmt.Println("Listening")

	clientSocket := syscall.RawSockaddrInet6{}
	clientSocketLength := C.sizeof_struct_sockaddr_in6
	newSockfd := C.zts_accept(sockfd, (* C.struct_sockaddr)(unsafe.Pointer(&clientSocket)), (* C.socklen_t)(unsafe.Pointer(&clientSocketLength)))
	validate(newSockfd, "ERROR on accept")
	fmt.Println("Accepted")

	clientIpAddress := make([]byte, C.ZT_MAX_IPADDR_LEN)
	C.inet_ntop(syscall.AF_INET6, unsafe.Pointer(&clientSocket.Addr), (* C.char)(unsafe.Pointer(&clientIpAddress[0])), C.ZT_MAX_IPADDR_LEN)
	fmt.Printf("Incoming connection from client having IPv6 address: %s\n", string(clientIpAddress[:C.ZT_MAX_IPADDR_LEN]))

	return int(newSockfd)
}
func main() {
	fmt.Println("Hello")

	C.zts_simple_start(C.CString("./zt"), C.CString(NETWORK_ID))

	ipv4Address := make([]byte, C.ZT_MAX_IPADDR_LEN)
	ipv6Address := make([]byte, C.ZT_MAX_IPADDR_LEN)

	C.zts_get_ipv4_address(C.CString(NETWORK_ID), (* C.char)(unsafe.Pointer(&ipv4Address[0])), C.ZT_MAX_IPADDR_LEN);
	fmt.Printf("ipv4 = %s \n", string(ipv4Address[:C.ZT_MAX_IPADDR_LEN]))

	C.zts_get_ipv6_address(C.CString(NETWORK_ID), (* C.char)(unsafe.Pointer(&ipv6Address[0])), C.ZT_MAX_IPADDR_LEN);
	fmt.Printf("ipv6 = %s \n", string(ipv6Address[:C.ZT_MAX_IPADDR_LEN]))

	sockfd := C.zts_socket(syscall.AF_INET6, syscall.SOCK_STREAM, 0)

	validate(sockfd, "Error in opening socket")

	if len(getOtherIP()) == 0 {
		newSockfd := bindAndListen(sockfd)

		//size_t readLength = read(from_fd, buffer, BUF_SIZE);
		//write(to_fd, buffer, readLength);

		packet := make([]byte, BUF_SIZE)
		go func() {
			for {
				plen, err := syscall.Read(newSockfd, packet)

				onError(err, "Error on reading from socket")

				//header, _ := ipv4.ParseHeader(packet[:plen])
				//fmt.Println("Sending to remote: %+v", header)
				fmt.Print(string(packet[:plen]))

				//sendRawMessage(peer, packet[:plen])
			}
		}()

	} else {

	}

	<-setupCleanUpOnInterrupt()
}
