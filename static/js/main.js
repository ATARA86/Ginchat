var ws;
var currentUser = null;
var currentToken = null;
var users = [];
var selectedImageFile = null;

function showToast(message, type) {
    var bgClass = type === 'success' ? 'bg-success' : (type === 'error' ? 'bg-danger' : 'bg-info');
    var html = '<div class="toast" role="alert">' +
        '<div class="toast-body ' + bgClass + ' text-white">' + message + '</div>' +
        '</div>';
    $('#toastContainer').html(html);
    $('.toast').toast('show');
}

function doLogin() {
    var name = $('#loginName').val();
    var password = $('#loginPassword').val();

    if (!name || !password) {
        showToast('请输入用户名和密码', 'error');
        return;
    }

    $.ajax({
        url: 'http://127.0.0.1:8081/users/login',
        type: 'POST',
        contentType: 'application/json',
        data: JSON.stringify({
            name: name,
            password: password
        }),
        success: function(res) {
            if (res.token) {
                currentToken = res.token;
                $.ajax({
                    url: 'http://127.0.0.1:8081/users/auth/me',
                    type: 'GET',
                    headers: {
                        'Authorization': 'Bearer ' + currentToken
                    },
                    success: function(userRes) {
                        if (userRes.code === 200) {
                            currentUser = userRes.data;
                            showToast('登录成功！', 'success');
                            showChatView();
                            loadUsers();
                            connectWebSocket();
                        }
                    },
                    error: function() {
                        showToast('获取用户信息失败', 'error');
                    }
                });
            } else {
                showToast(res.message || '登录失败', 'error');
            }
        },
        error: function(err) {
            showToast('登录失败: ' + (err.responseJSON?.message || err.statusText), 'error');
        }
    });
}

function doRegister() {
    var name = $('#regName').val();
    var password = $('#regPassword').val();
    var repassword = $('#regRepassword').val();
    var phone = $('#regPhone').val();
    var email = $('#regEmail').val();

    if (!name || !password) {
        showToast('请输入用户名和密码', 'error');
        return;
    }

    if (password !== repassword) {
        showToast('两次密码不一致', 'error');
        return;
    }

    $.ajax({
        url: 'http://127.0.0.1:8081/users',
        type: 'POST',
        data: {
            name: name,
            password: password,
            repassword: repassword,
            phone: phone,
            email: email
        },
        success: function(res) {
            if (res.code === 200) {
                showToast('注册成功！请登录', 'success');
                $('#authTabs a[href="#loginTab"]').tab('show');
                $('#loginName').val(name);
            } else {
                showToast(res.message || '注册失败', 'error');
            }
        },
        error: function(err) {
            showToast('注册失败: ' + (err.responseJSON?.message || err.statusText), 'error');
        }
    });
}

function loadUsers() {
    $.ajax({
        url: 'http://127.0.0.1:8081/users/auth/friends',
        type: 'GET',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                users = res.data || [];
                renderUserList();
            }
        },
        error: function() {
            showToast('加载好友列表失败', 'error');
        }
    });
}

function renderUserList() {
    var html = '';
    var selectHtml = '<option value="">选择用户</option>';

    users.forEach(function(user) {
        if (user.ID !== currentUser.ID) {
            html += '<div class="user-item" onclick="selectUser(' + user.ID + ', \'' + user.Name + '\')">' +
                '<div class="d-flex align-items-center">' +
                '<div class="user-card me-2">' + (user.Name ? user.Name.charAt(0).toUpperCase() : '?') + '</div>' +
                '<div>' +
                '<div class="fw-bold">' + user.Name + '</div>' +
                '<small class="text-muted">ID: ' + user.ID + '</small>' +
                '</div>' +
                '</div>' +
                '</div>';
            selectHtml += '<option value="' + user.ID + '">' + user.Name + ' (ID:' + user.ID + ')</option>';
        }
    });

    $('#userList').html(html);
    $('#targetSelect').html(selectHtml);
}

function selectUser(userId, userName) {
    $('#userList .user-item').removeClass('active');
    $('#targetSelect').val(userId);
}

function showChatView() {
    $('#authView').hide();
    $('#chatView').show();
    $('#currentUserBadge').text('当前用户: ' + currentUser.Name);
    loadUserGroups();
}

function logout() {
    if (ws) {
        ws.close();
    }
    currentUser = null;
    currentToken = null;
    $('#chatView').hide();
    $('#authView').show();
    $('#messageList').html('');
}

function connectWebSocket() {
    ws = new WebSocket('ws://127.0.0.1:8081/ws/user?userID=' + currentUser.ID + '&token=' + currentToken);

    ws.onopen = function() {
        $('#connectionStatus').text('已连接');
        $('#connectionStatus').removeClass('bg-danger').addClass('bg-success');
        addSystemMessage('欢迎 ' + currentUser.Name + ' 进入聊天系统！');
    };

    ws.onmessage = function(event) {
        try {
            var data = JSON.parse(event.data);
            if (data.Type === 1) {
                if (data.Media === 2 && data.Pic) {
                    addImageMessage(data.FormID, data.Pic, 'received');
                } else {
                    addMessage(data.FormID, data.Content, 'received');
                }
            } else if (data.Type === 2) {
                displayGroupMessage(data, false);
            }
        } catch (e) {
            console.log('消息解析失败:', e);
        }
    };

    ws.onerror = function() {
        $('#connectionStatus').text('连接失败');
        $('#connectionStatus').removeClass('bg-success').addClass('bg-danger');
    };

    ws.onclose = function() {
        $('#connectionStatus').text('未连接');
        $('#connectionStatus').removeClass('bg-success').addClass('bg-danger');
        addSystemMessage('连接已断开');
    };
}

function sendMessage() {
    var targetId = $('#targetSelect').val();

    if (!targetId) {
        showToast('请选择发送对象', 'error');
        return;
    }

    if (!ws || ws.readyState != WebSocket.OPEN) {
        showToast('WebSocket未连接', 'error');
        return;
    }

    if (selectedImageFile) {
        sendImageMessage(parseInt(targetId));
        return;
    }

    var content = $('#messageInput').val();

    if (!content) {
        showToast('请输入消息内容', 'error');
        return;
    }

    var msg = {
        FormID: currentUser.ID,
        TargetID: parseInt(targetId),
        Type: 1,
        Media: 1,
        Content: content
    };

    ws.send(JSON.stringify(msg));
    addMessage(currentUser.ID, content, 'sent');
    $('#messageInput').val('');
}

function handleImageSelect(event) {
    var file = event.target.files[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
        showToast('请选择图片文件', 'error');
        return;
    }

    if (file.size > 5 * 1024 * 1024) {
        showToast('图片大小不能超过5MB', 'error');
        return;
    }

    selectedImageFile = file;
    showToast('已选择图片: ' + file.name + '，点击发送', 'info');
}

function sendImageMessage(targetId) {
    if (!selectedImageFile) return;

    var reader = new FileReader();
    reader.onload = function(e) {
        var base64Data = e.target.result;

        var msg = {
            FormID: currentUser.ID,
            TargetID: targetId,
            Type: 1,
            Media: 2,
            Content: '',
            Pic: base64Data
        };

        ws.send(JSON.stringify(msg));
        addImageMessage(currentUser.ID, base64Data, 'sent');
        $('#messageInput').val('');
        selectedImageFile = null;
        $('#imageInput').val('');
    };
    reader.readAsDataURL(selectedImageFile);
}

function addImageMessage(userId, imageData, type) {
    var userName = userId === currentUser.ID ? '我' : (users.find(u => u.ID === userId)?.Name || '用户' + userId);
    var html = '<div class="message-item ' + type + '">' +
        '<div class="message-bubble">' +
        '<img src="' + imageData + '" style="max-width: 200px; border-radius: 8px;">' +
        '<div class="message-info">' + userName + ' - ' + new Date().toLocaleTimeString() + '</div>' +
        '</div>' +
        '</div>';
    $('#messageList').append(html);
    scrollToBottom();
}

function addMessage(userId, content, type) {
    var userName = userId === currentUser.ID ? '我' : (users.find(u => u.ID === userId)?.Name || '用户' + userId);
    var html = '<div class="message-item ' + type + '">' +
        '<div class="message-bubble">' +
        content +
        '<div class="message-info">' + userName + ' - ' + new Date().toLocaleTimeString() + '</div>' +
        '</div>' +
        '</div>';
    $('#messageList').append(html);
    scrollToBottom();
}

function addSystemMessage(content) {
    var html = '<div class="text-center text-muted my-3">' +
        '<small><i class="fas fa-info-circle"></i> ' + content + '</small>' +
        '</div>';
    $('#messageList').append(html);
    scrollToBottom();
}

function scrollToBottom() {
    $('#messageList').scrollTop($('#messageList')[0].scrollHeight);
}

function handleKeyPress(event) {
    if (event.key === 'Enter') {
        sendMessage();
    }
}

var allUsers = [];

function loadAllUsers() {
    $.ajax({
        url: 'http://127.0.0.1:8081/users',
        type: 'GET',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                allUsers = res.data || [];
                renderFriendSelect();
            }
        },
        error: function() {
            showToast('加载用户列表失败', 'error');
        }
    });
}

function renderFriendSelect() {
    var currentFriendIds = users.map(u => u.ID);
    var availableUsers = allUsers.filter(u => u.ID !== currentUser.ID && !currentFriendIds.includes(u.ID));

    var html = '<option value="">选择用户</option>';
    availableUsers.forEach(function(user) {
        html += '<option value="' + user.ID + '" data-name="' + user.Name + '">' + user.Name + ' (ID:' + user.ID + ')</option>';
    });
    $('#friendSelect').html(html);

    if (availableUsers.length === 0) {
        html = '<option value="">没有可添加的用户</option>';
        $('#friendSelect').html(html);
    }
}

function showAddFriendModal() {
    loadAllUsers();
    var modal = new bootstrap.Modal(document.getElementById('addFriendModal'));
    modal.show();
}

function doAddFriend() {
    var targetId = $('#friendSelect').val();
    if (!targetId) {
        showToast('请选择要添加的好友', 'error');
        return;
    }

    $.ajax({
        url: 'http://127.0.0.1:8081/users/auth/add-friend?target_id=' + targetId,
        type: 'POST',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                showToast('添加成功！', 'success');
                bootstrap.Modal.getInstance(document.getElementById('addFriendModal')).hide();
                loadUsers();
            } else {
                showToast(res.msg || '添加失败', 'error');
            }
        },
        error: function(err) {
            showToast('添加失败: ' + (err.responseJSON?.msg || err.statusText), 'error');
        }
    });
}

function showEditProfileModal() {
    $('#editName').val(currentUser.Name);
    $('#editPassword').val('');
    $('#editPhone').val(currentUser.Phone || '');
    $('#editEmail').val(currentUser.Email || '');
    var modal = new bootstrap.Modal(document.getElementById('editProfileModal'));
    modal.show();
}

function doUpdateProfile() {
    var name = $('#editName').val().trim();
    var password = $('#editPassword').val();
    var phone = $('#editPhone').val().trim();
    var email = $('#editEmail').val().trim();

    if (!name) {
        showToast('用户名不能为空', 'error');
        return;
    }

    var url = 'http://127.0.0.1:8081/users/auth/' + currentUser.ID;
    var params = [];
    if (name) params.push('name=' + encodeURIComponent(name));
    if (password) params.push('password=' + encodeURIComponent(password));
    if (phone) params.push('phone=' + encodeURIComponent(phone));
    if (email) params.push('email=' + encodeURIComponent(email));

    if (params.length > 0) {
        url += '?' + params.join('&');
    }

    $.ajax({
        url: url,
        type: 'PUT',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                showToast('更新成功！', 'success');
                bootstrap.Modal.getInstance(document.getElementById('editProfileModal')).hide();
                currentUser.Name = name;
                currentUser.Phone = phone;
                currentUser.Email = email;
                $('#currentUserBadge').text('当前用户: ' + name);
            } else {
                showToast(res.msg || '更新失败', 'error');
            }
        },
        error: function(err) {
            showToast('更新失败: ' + (err.responseJSON?.msg || err.statusText), 'error');
        }
    });
}

var userGroups = [];
var currentGroup = null;
var groupMembers = [];

function loadUserGroups() {
    $.ajax({
        url: 'http://127.0.0.1:8081/groups/user/' + currentUser.ID,
        type: 'GET',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                userGroups = res.data || [];
                renderGroupSelect();
            }
        },
        error: function() {
            showToast('加载群列表失败', 'error');
        }
    });
}

function renderGroupSelect() {
    var html = '<option value="">选择群聊</option>';
    userGroups.forEach(function(group) {
        html += '<option value="' + group.ID + '">' + group.Name + '</option>';
    });
    if ($('#groupSelect').length > 0) {
        $('#groupSelect').html(html);
    }
}

function showCreateGroupModal() {
    var modal = new bootstrap.Modal(document.getElementById('createGroupModal'));
    modal.show();
}

function doCreateGroup() {
    var name = $('#groupName').val().trim();
    var desc = $('#groupDesc').val().trim();

    if (!name) {
        showToast('群名称不能为空', 'error');
        return;
    }

    $.ajax({
        url: 'http://127.0.0.1:8081/groups',
        type: 'POST',
        contentType: 'application/json',
        data: JSON.stringify({
            name: name,
            desc: desc,
            owner: currentUser.ID
        }),
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                showToast('创建群成功！', 'success');
                bootstrap.Modal.getInstance(document.getElementById('createGroupModal')).hide();
                $('#groupName').val('');
                $('#groupDesc').val('');
                loadUserGroups();
            } else {
                showToast(res.msg || '创建失败', 'error');
            }
        },
        error: function(err) {
            showToast('创建失败: ' + (err.responseJSON?.msg || err.statusText), 'error');
        }
    });
}

function showJoinGroupModal() {
    loadAllUsersForGroup();
    var modal = new bootstrap.Modal(document.getElementById('joinGroupModal'));
    modal.show();
}

function loadAllUsersForGroup() {
    $.ajax({
        url: 'http://127.0.0.1:8081/users',
        type: 'GET',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                var users = res.data || [];
                var html = '<option value="">选择用户</option>';
                users.forEach(function(user) {
                    if (user.ID !== currentUser.ID) {
                        html += '<option value="' + user.ID + '">' + user.Name + ' (ID:' + user.ID + ')</option>';
                    }
                });
                $('#joinGroupUserSelect').html(html);
            }
        },
        error: function() {
            showToast('加载用户列表失败', 'error');
        }
    });
}

function doJoinGroup() {
    var groupId = $('#joinGroupId').val().trim();
    var userId = $('#joinGroupUserSelect').val();

    if (!groupId) {
        showToast('请输入群ID', 'error');
        return;
    }

    if (!userId) {
        showToast('请选择要拉入群的用户', 'error');
        return;
    }

    $.ajax({
        url: 'http://127.0.0.1:8081/groups/add',
        type: 'POST',
        contentType: 'application/json',
        data: JSON.stringify({
            group_id: parseInt(groupId),
            user_id: parseInt(userId)
        }),
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                showToast('加入群成功！', 'success');
                bootstrap.Modal.getInstance(document.getElementById('joinGroupModal')).hide();
                loadUserGroups();
            } else {
                showToast(res.msg || '加入失败', 'error');
            }
        },
        error: function(err) {
            showToast('加入失败: ' + (err.responseJSON?.msg || err.statusText), 'error');
        }
    });
}

function selectGroup(groupId) {
    if (!groupId) {
        currentGroup = null;
        groupMembers = [];
        return;
    }

    currentGroup = userGroups.find(g => g.ID === parseInt(groupId));
    loadGroupMembers(groupId);
    loadGroupMessages(groupId);
}

function loadGroupMembers(groupId) {
    $.ajax({
        url: 'http://127.0.0.1:8081/groups/members?group_id=' + groupId,
        type: 'GET',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                groupMembers = res.data || [];
                renderGroupMembers();
            }
        },
        error: function() {
            showToast('加载群成员失败', 'error');
        }
    });
}

function renderGroupMembers() {
    var html = '<small class="text-muted">群成员: ';
    groupMembers.forEach(function(member, index) {
        html += member.Name;
        if (index < groupMembers.length - 1) html += ', ';
    });
    html += '</small>';
    $('#groupMembersDisplay').html(html);
}

function loadGroupMessages(groupId) {
    $.ajax({
        url: 'http://127.0.0.1:8081/groups/messages?group_id=' + groupId,
        type: 'GET',
        headers: {
            'Authorization': 'Bearer ' + currentToken
        },
        success: function(res) {
            if (res.code === 200) {
                var messages = res.data || [];
                messages.reverse();
                messages.forEach(function(msg) {
                    displayGroupMessage(msg, false);
                });
            }
        },
        error: function() {
            showToast('加载群消息失败', 'error');
        }
    });
}

function displayGroupMessage(msg, isSent) {
    var senderName = msg.FormID === currentUser.ID ? '我' : (groupMembers.find(m => m.ID === msg.FormID)?.Name || '用户' + msg.FormID);
    var type = isSent ? 'sent' : 'received';

    if (msg.Media === 2 && msg.Pic) {
        var html = '<div class="message-item ' + type + '">' +
            '<div class="message-bubble">' +
            '<img src="' + msg.Pic + '" style="max-width: 200px; border-radius: 8px;">' +
            '<div class="message-info">' + senderName + ' - ' + new Date(msg.CreatedAt).toLocaleTimeString() + '</div>' +
            '</div>' +
            '</div>';
        $('#messageList').append(html);
    } else {
        addMessage(msg.FormID, msg.Content, type);
    }
    scrollToBottom();
}

function sendGroupChatMessage() {
    if (!currentGroup) {
        showToast('请先选择一个群', 'error');
        return;
    }

    if (!ws || ws.readyState != WebSocket.OPEN) {
        showToast('WebSocket未连接', 'error');
        return;
    }

    var content = $('#messageInput').val();
    if (!content) {
        showToast('请输入消息内容', 'error');
        return;
    }

    var msg = {
        FormID: currentUser.ID,
        TargetID: currentGroup.ID,
        Type: 2,
        Media: 1,
        Content: content
    };

    ws.send(JSON.stringify(msg));
    displayGroupMessage({
        FormID: currentUser.ID,
        Content: content,
        Media: 1,
        CreatedAt: new Date()
    }, true);
    $('#messageInput').val('');
}