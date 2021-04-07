#!/bin/bash
#sudo apt install inotify-tools
# usage:
#   nohub ./shareCloudInitFinalStatusAsVirtualBoxGuestProperties.sh &>/dev/null &

# will complain if file does not exists
#cloud_init_instance_finished_dir=/tmp/00inotify/
cloud_init_instance_finished_dir=/var/lib/cloud/instance
cloud_init_instance_finished_file=$${cloud_init_instance_finished_dir}/boot-finished
timeout_secs=300

doShareCloudInitStatusAsVirtualBoxGuest() {
    VBoxControl guestproperty set "/VirtualBox/GuestInfo/CloudInit/Status" "$(cloud-init status)"

    if [ -e "$${cloud_init_instance_finished_file}" ]
    then
        VBoxControl guestproperty set "/VirtualBox/GuestInfo/CloudInit/Finished" "True"
    else 
        VBoxControl guestproperty set "/VirtualBox/GuestInfo/CloudInit/Finished" "False"
    fi
}

waitForCloudInitFinished() { 
inotifywait -t $${timeout_secs} -e create -e moved_to -e modify $${cloud_init_instance_finished_dir} |
    while read dir action file; do
        echo "The file '$${file}' appeared in directory '$${dir}' via '$${action}'"
        [ "$${file}" = "boot-finished" ] && doShareCloudInitStatusAsVirtualBoxGuest | true
        [ "$${file}" = "boot-finished" ] && exit 0
        continue
        # do something with the file
    done
}

export timeout_secs
export cloud_init_instance_finished_dir
export cloud_init_instance_finished_file
export -f waitForCloudInitFinished
export -f doShareCloudInitStatusAsVirtualBoxGuest

if [ -e "$${cloud_init_instance_finished_file}" ]
then
    echo "$${cloud_init_instance_finished_file} already exists just sharing cloud-init status as virtual box guest-property"
    doShareCloudInitStatusAsVirtualBoxGuest
else 
    nohup timeout $${timeout_secs}s bash -c "waitForCloudInitFinished" &>/var/log/cloud-init-virtualbox.log &
fi

exit 0
