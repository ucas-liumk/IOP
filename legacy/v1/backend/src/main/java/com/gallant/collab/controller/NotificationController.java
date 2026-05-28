package com.gallant.collab.controller;

import com.gallant.collab.service.NotificationService;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/notifications")
@RequiredArgsConstructor
public class NotificationController {

    private final NotificationService service;

    @GetMapping("/unread")
    public NotificationService.NotificationDigest unread() {
        return service.fetch();
    }

    @PostMapping("/messages/{id}/read")
    public void markRead(@PathVariable Long id) {
        service.markMessageRead(id);
    }
}
