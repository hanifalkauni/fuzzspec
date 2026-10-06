<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;

class OrderController extends Controller
{
    public function show($id)
    {
        // Bug: Unhandled fatal error on negative ID / SQL injection string
        if (!is_numeric($id) || $id < 0) {
            throw new \ErrorException("Fatal error: Uncaught TypeError: OrderRepository::find(): Argument #1 (\$id) must be of type int\nStack trace:\n#0 /var/www/app/Http/Controllers/OrderController.php(14): OrderRepository->find('$id')");
        }

        return response()->json(['id' => (int)$id, 'status' => 'COMPLETED']);
    }
}
