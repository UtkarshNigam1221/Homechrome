# Notification Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for the Notification Lambda service, illustrating the interactions between components for notification creation, delivery, and management.

---

## 1. Create Notification (Admin)

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Admin  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ POST /notifications                   │                      │
     │ {type, title, message, recipients}    │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │ (Admin role)        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ CreateNotification()│                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Validate input       │
     │                 │                     │──────────┐           │
     │                 │                     │          │           │
     │                 │                     │<─────────┘           │
     │                 │                     │                      │
     │                 │                     │ Get recipient list   │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ User list            │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │                     │ Create notification  │
     │                 │                     │ records (batch)      │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Success              │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ Notification created│                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 201 Created     │                     │                      │
     │ {id, status}    │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 2. System-Triggered Notification (Event-Driven)

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Order Lambda │     │   EventBridge    │     │ Notification Svc │     │  DynamoDB    │
└──────┬───────┘     └────────┬─────────┘     └────────┬─────────┘     └──────┬───────┘
       │                      │                        │                      │
       │ Publish Event        │                        │                      │
       │ {ORDER_CREATED}      │                        │                      │
       │─────────────────────>│                        │                      │
       │                      │                        │                      │
       │                      │ Trigger Lambda         │                      │
       │                      │───────────────────────>│                      │
       │                      │                        │                      │
       │                      │                        │ Parse event          │
       │                      │                        │──────────┐           │
       │                      │                        │          │           │
       │                      │                        │<─────────┘           │
       │                      │                        │                      │
       │                      │                        │ Get notification     │
       │                      │                        │ template             │
       │                      │                        │─────────────────────>│
       │                      │                        │                      │
       │                      │                        │ Template             │
       │                      │                        │<─────────────────────│
       │                      │                        │                      │
       │                      │                        │ Get user preferences │
       │                      │                        │─────────────────────>│
       │                      │                        │                      │
       │                      │                        │ Preferences          │
       │                      │                        │<─────────────────────│
       │                      │                        │                      │
       │                      │                        │ Create notification  │
       │                      │                        │─────────────────────>│
       │                      │                        │                      │
       │                      │                        │ Success              │
       │                      │                        │<─────────────────────│
       │                      │                        │                      │
```

---

## 3. Get User Notifications

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ GET /notifications                    │                      │
     │ ?status=unread&limit=20               │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │ Extract user_id     │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ GetUserNotifications│                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Query by user_id     │
     │                 │                     │ (filter: unread)     │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Notification list    │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │                     │ Sort by created_at   │
     │                 │                     │──────────┐           │
     │                 │                     │          │           │
     │                 │                     │<─────────┘           │
     │                 │                     │                      │
     │                 │ Notifications       │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 200 OK          │                     │                      │
     │ {notifications, │                     │                      │
     │  unread_count}  │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 4. Mark Notification as Read

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ PATCH /notifications/{id}/read        │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ MarkAsRead()        │                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Get notification     │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Notification         │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │                     │ Verify ownership     │
     │                 │                     │──────────┐           │
     │                 │                     │          │           │
     │                 │                     │<─────────┘           │
     │                 │                     │                      │
     │                 │                     │ Update read_at       │
     │                 │                     │ status=read          │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Success              │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ Updated             │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 200 OK          │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 5. Mark All Notifications as Read

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ POST /notifications/mark-all-read     │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ MarkAllAsRead()     │                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Query unread         │
     │                 │                     │ notifications        │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Unread list          │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │                     │ Batch update         │
     │                 │                     │ (set read_at)        │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Success              │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ {updated_count}     │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 200 OK          │                     │                      │
     │ {count: 5}      │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 6. Get Unread Count

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ GET /notifications/unread-count       │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ GetUnreadCount()    │                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Count query          │
     │                 │                     │ (status=unread)      │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Count: 5             │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ Count: 5            │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 200 OK          │                     │                      │
     │ {unread: 5}     │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 7. Delete Notification

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ DELETE /notifications/{id}            │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ DeleteNotification()│                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Get notification     │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Notification         │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │                     │ Verify ownership     │
     │                 │                     │──────────┐           │
     │                 │                     │          │           │
     │                 │                     │<─────────┘           │
     │                 │                     │                      │
     │                 │                     │ Delete record        │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Success              │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ Deleted             │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 204 No Content  │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 8. Send Email Notification

```
┌──────────────────┐     ┌──────────────┐     ┌─────────┐     ┌──────────────┐
│ Notification Svc │     │  DynamoDB    │     │   SES   │     │  CloudWatch  │
└────────┬─────────┘     └──────┬───────┘     └────┬────┘     └──────┬───────┘
         │                      │                  │                  │
         │ Get pending          │                  │                  │
         │ email notifications  │                  │                  │
         │─────────────────────>│                  │                  │
         │                      │                  │                  │
         │ Notification list    │                  │                  │
         │<─────────────────────│                  │                  │
         │                      │                  │                  │
         │ For each notification│                  │                  │
         │──────────┐           │                  │                  │
         │          │           │                  │                  │
         │<─────────┘           │                  │                  │
         │                      │                  │                  │
         │ Get user email       │                  │                  │
         │─────────────────────>│                  │                  │
         │                      │                  │                  │
         │ User data            │                  │                  │
         │<─────────────────────│                  │                  │
         │                      │                  │                  │
         │ Send email           │                  │                  │
         │─────────────────────────────────────────>│                  │
         │                      │                  │                  │
         │ Email sent           │                  │                  │
         │<─────────────────────────────────────────│                  │
         │                      │                  │                  │
         │ Update delivery      │                  │                  │
         │ status               │                  │                  │
         │─────────────────────>│                  │                  │
         │                      │                  │                  │
         │ Success              │                  │                  │
         │<─────────────────────│                  │                  │
         │                      │                  │                  │
         │ Log delivery         │                  │                  │
         │────────────────────────────────────────────────────────────>│
         │                      │                  │                  │
```

---

## 9. Error Handling - Notification Not Found

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ GET /notifications/{invalid_id}       │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ GetNotification()   │                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Query by ID          │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Not found            │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ Error: Not found    │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 404 Not Found   │                     │                      │
     │ {error}         │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

---

## 10. Update Notification Preferences

```
┌─────────┐     ┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Client  │     │   API GW    │     │ Notification Svc │     │  DynamoDB    │
└────┬────┘     └──────┬──────┘     └────────┬─────────┘     └──────┬───────┘
     │                 │                     │                      │
     │ PUT /notifications/preferences        │                      │
     │ {email: true, push: false, ...}       │                      │
     │────────────────>│                     │                      │
     │                 │                     │                      │
     │                 │ Validate JWT        │                      │
     │                 │──────────┐          │                      │
     │                 │          │          │                      │
     │                 │<─────────┘          │                      │
     │                 │                     │                      │
     │                 │ UpdatePreferences() │                      │
     │                 │────────────────────>│                      │
     │                 │                     │                      │
     │                 │                     │ Validate preferences │
     │                 │                     │──────────┐           │
     │                 │                     │          │           │
     │                 │                     │<─────────┘           │
     │                 │                     │                      │
     │                 │                     │ Update user          │
     │                 │                     │ preferences          │
     │                 │                     │─────────────────────>│
     │                 │                     │                      │
     │                 │                     │ Success              │
     │                 │                     │<─────────────────────│
     │                 │                     │                      │
     │                 │ Updated preferences │                      │
     │                 │<────────────────────│                      │
     │                 │                     │                      │
     │ 200 OK          │                     │                      │
     │ {preferences}   │                     │                      │
     │<────────────────│                     │                      │
     │                 │                     │                      │
```

