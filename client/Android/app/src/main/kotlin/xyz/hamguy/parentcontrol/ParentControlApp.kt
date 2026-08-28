package xyz.hamguy.parentcontrol

import android.app.Application
import xyz.hamguy.parentcontrol.repository.ParentControlRepository

class ParentControlApp : Application() {
    lateinit var repository: ParentControlRepository
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this
        repository = ParentControlRepository()
    }

    companion object {
        lateinit var instance: ParentControlApp
            private set
    }
}
