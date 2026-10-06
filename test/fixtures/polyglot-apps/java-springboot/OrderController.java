package com.example.demo.controller;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/orders")
public class OrderController {

    @GetMapping("/{id}")
    public ResponseEntity<?> getOrder(@PathVariable("id") Long id) {
        if (id == null || id > 1000000L) {
            throw new NullPointerException("java.lang.NullPointerException: Cannot invoke 'Order.getId()' because order is null\n\tat com.example.demo.controller.OrderController.getOrder(OrderController.java:12)");
        }
        return ResponseEntity.ok("{\"order_id\": " + id + ", \"status\": \"ACTIVE\"}");
    }
}
